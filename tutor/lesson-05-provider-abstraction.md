# Lesson 5: Provider 抽象

## 这一课学完你能做什么
你可以解释 Bifrost 为什么能把 OpenAI、Anthropic、Groq、Gemini 等不同厂商统一到同一个执行引擎里；你也能判断一个新 provider 应该走“OpenAI-compatible 复用模式”还是“非兼容 provider 独立转换模式”，并为自己的 `bifrost-lite` 加上第二个 provider。

## 这一层的职责
provider abstraction 的职责很简单，但非常关键：

1. 对上给 core 一个统一接口
2. 对下吸收不同模型厂商的协议差异
3. 让 transport/core 不必知道各家 API 的细节
4. 为 fallback、retry、plugin、tracing 提供统一执行面

一句话说：这一层负责“把异构模型厂商变成同构能力”。

## 它在系统里的位置

上游：
- `Bifrost` core
- `handleProviderRequest(...)`
- `requestWorker(...)`

当前层：
- `schemas.Provider` 接口
- `core/providers/<name>/`
- request/response converters

下游：
- OpenAI API
- Anthropic API
- Bedrock API
- 其他 provider HTTP 协议

所以 provider abstraction 不是给外部调用的，它是给 core 用的。

## 关键执行链路

### 1. core 只认 `Provider` 接口
统一接口定义在 [`core/schemas/provider.go`](/mnt/windows/source/go/bifrost/core/schemas/provider.go:553)。

你会马上看到一个现实：

这个接口很大，非常大。

它覆盖了：
- `ListModels`
- `TextCompletion`
- `ChatCompletion`
- `Responses`
- `Embedding`
- `Rerank`
- `Speech`
- `Transcription`
- `ImageGeneration`
- `VideoGeneration`
- `Batch*`
- `File*`
- `Container*`
- `Passthrough*`

这告诉你一件很重要的事：

Bifrost 不是只做聊天接口，而是在试图把“LLM provider 能提供的能力面”统一成一个网关能力面。

### 2. core 调 provider 时根本不关心是哪一家
在 Lesson 4 里你已经见过 [`handleProviderRequest(...)`](/mnt/windows/source/go/bifrost/core/bifrost.go:5041)。

比如 chat 请求的分发本质上就是：

```go
provider.ChatCompletion(req.Context, key, req.BifrostRequest.ChatRequest)
```

这里没有：

- OpenAI 特判
- Anthropic 特判
- Gemini 特判

这就是 provider abstraction 的价值：让 core 完全不知道底层协议差异。

### 3. OpenAI provider 是“参考实现”
OpenAI 实现是整个系统里最有代表性的 provider 之一，入口在 [`core/providers/openai/openai.go`](/mnt/windows/source/go/bifrost/core/providers/openai/openai.go:21)。

这个 provider 做了几件典型的事：

1. 在 `NewOpenAIProvider(...)` 里初始化 `fasthttp.Client`
2. 配置 timeout、连接池、TLS、proxy
3. 默认 `BaseURL = https://api.openai.com`
4. 把每一种请求都交给 `HandleOpenAI*` 系列函数

比如：
- `ChatCompletion()` -> `HandleOpenAIChatCompletionRequest(...)`
- `TextCompletion()` -> `HandleOpenAITextCompletionRequest(...)`
- `ListModels()` -> `HandleOpenAIListModelsRequest(...)`

这说明 OpenAI provider 既是一个具体实现，也是很多兼容 provider 的“基础模板”。

### 4. OpenAI-compatible provider 大量复用 openai 逻辑
Groq 是最典型的例子，代码在 [`core/providers/groq/groq.go`](/mnt/windows/source/go/bifrost/core/providers/groq/groq.go:14)。

你会发现它的 `ChatCompletion()` 基本就是：

1. 构造自己的 base URL
2. 复用 `openai.HandleOpenAIChatCompletionRequest(...)`

`ChatCompletionStream()` 也类似，直接走 `openai.HandleOpenAIChatCompletionStreaming(...)`。

这类 provider 的特点是：

- 请求路径和认证方式接近 OpenAI
- 请求/响应结构基本兼容
- 少量字段兼容性差异可以在 shared converter 里处理

所以这类 provider 的实现成本很低，关键是“站在 OpenAI 抽象之上复用”。

### 5. 非兼容 provider 要自己做 request/response 转换
Anthropic 是另一种路线，入口在 [`core/providers/anthropic/anthropic.go`](/mnt/windows/source/go/bifrost/core/providers/anthropic/anthropic.go:432)。

以 `ChatCompletion()` 为例，它做的路径是：

1. `ToAnthropicChatRequest(ctx, request)` 把 Bifrost request 转成 Anthropic request
2. `AddMissingBetaHeadersToContext(...)` 补 Anthropic 特有 header
3. `provider.completeRequest(...)` 发 HTTP 请求
4. 把原始响应解码为 Anthropic response struct
5. `response.ToBifrostChatResponse(ctx)` 再转回 Bifrost response

这个模式和 OpenAI-compatible provider 完全不同：

- 需要独立 converter
- 需要独立 response parser
- 需要独立 stream 事件转换

这才是统一抽象真正辛苦的部分。

### 6. converter 是 provider 抽象的真正核心
OpenAI chat 转换可以看 [`core/providers/openai/chat.go`](/mnt/windows/source/go/bifrost/core/providers/openai/chat.go:10)。

这里有两类非常经典的函数：

- `ToBifrostChatRequest(...)`
- `ToOpenAIChatRequest(...)`

Anthropic 也有对应思路：

- `ToAnthropicChatRequest(...)`
- `ToBifrostChatResponse(...)`

这说明 provider abstraction 的本质不是“把 HTTP 发出去”，而是：

```text
Bifrost schema <-> Provider-native schema
```

只要这组转换做对了，core 就能稳定工作。

### 7. OpenAI 共享 converter 里还在做 provider-specific 兼容修正
即使都是 OpenAI-compatible provider，也不是完全零差异。

在 [`ToOpenAIChatRequest(...)`](/mnt/windows/source/go/bifrost/core/providers/openai/chat.go:24) 里，你能看到很多兼容处理：

- `xAI` 要过滤部分 OpenAI 参数
- `Gemini` 要去掉不支持字段
- `Mistral` 要把 `max_completion_tokens` 改成 `max_tokens`
- `Fireworks` 要保留自己的 cache/prediction 字段

这说明 “OpenAI-compatible” 不是“完全一样”，而是“足够像，可以共用主干逻辑，再在边角处做修正”。

## 为什么这是项目亮点

### 1. 复杂性被成功压进 provider 层
统一多 provider 最大的工程价值，不在于“支持很多 logo”，而在于：

- transport 不需要写 N 份
- core 不需要写 N 份
- fallback/retry/plugin/tracing 只写一份

而这些都建立在 provider 层吸收了协议差异。

这就是非常典型的“复杂性下沉”。

### 2. OpenAI provider 被设计成“共享内核”
从 `Groq` 这种实现方式你能看出，Bifrost 不是每个 provider 都从零写。

它把 OpenAI provider 做成了：

- 自己可用
- 别人可复用
- 共享 handler 函数
- 共享 converter 主干

这是一种非常高质量的工程抽象。

### 3. 大接口不是为了炫，而是产品范围决定的
`Provider` 接口这么大，并不是作者不会拆分，而是产品本身要统一：

- chat
- responses
- embedding
- audio
- images
- files
- batches
- containers

这代表 Bifrost 的目标不是“聊天代理”，而是“统一 AI provider gateway”。

### 4. provider 分类策略非常适合面试表达
你完全可以把 Bifrost 的 provider 设计讲成两类：

1. OpenAI-compatible providers
2. Non-OpenAI-compatible providers

然后补一句：

“前者复用 shared OpenAI handler/converter，后者实现独立 request/response normalization。”

这句话会非常像一个做过网关系统的人说出来的。

## Bifrost 设计 takeaway

### 1. 不要把 provider 抽象成“只是一层 SDK 包装”
如果你只是写：

```go
type OpenAIProvider struct{}
type AnthropicProvider struct{}
```

然后每个 handler 手调不同 SDK，那不叫真正抽象。

真正的抽象要求：

- 有统一 request schema
- 有统一 response schema
- core 只调 interface
- 差异在 provider 层吃掉

### 2. 第二个 provider 才是抽象是否成立的试金石
你做 `bifrost-lite` 时，第一家 provider 通常都很好写。

真正的问题出在第二家：

- 字段不一样怎么办
- streaming 格式不一样怎么办
- 错误格式不一样怎么办

所以 Lesson 5 的重点不是“再接个 API”，而是“让第二家 provider 仍然走同一条核心执行链路”。

### 3. 先区分“兼容型”和“非兼容型”
这是你自己实现时最重要的策略判断。

如果第二家 provider 足够像 OpenAI：
- 复用 OpenAI request/response schema
- 只改 base URL / auth / 少数字段

如果第二家 provider 明显不一样：
- 单独定义 provider-native request/response
- 写专门 converter
- 让 core 仍然只看统一 schema

## 我自己这一课要实现什么

### 这节课你的 bifrost-lite 目标
在已有一个 provider 的基础上，加第二个 provider，并验证你的抽象不是假的。

### 推荐你怎么做

#### 路线 A：先做两个 OpenAI-compatible provider
最容易成功。

例如：
- `openai`
- `groq` 或 `openrouter`

你的实现方式可以是：

1. 定义统一 `Provider` 接口
2. 抽一个共享 `OpenAICompatibleProvider` 基类或 helper
3. 第二个 provider 只改 `BaseURL`、provider name、少数字段差异

#### 路线 B：再做一个非兼容 provider
更像 Bifrost，但工作量明显大。

例如：
- `openai`
- `anthropic`

这时你要自己写：

1. `ToAnthropicRequest(...)`
2. `ToBifrostResponse(...)`
3. streaming event 转换

如果你时间有限，建议先走路线 A，再走路线 B。

### 这一课你的最小文件结构

```text
internal/providers/provider.go
internal/providers/openai/openai.go
internal/providers/groq/groq.go
```

或者如果你想挑战非兼容：

```text
internal/providers/provider.go
internal/providers/openai/openai.go
internal/providers/anthropic/anthropic.go
internal/providers/anthropic/types.go
internal/providers/anthropic/convert.go
```

### 这一课的完成标准
你要做到：

1. core 只依赖 `Provider` 接口
2. 第二个 provider 能走同一条 `Gateway -> queue -> worker -> provider` 链路
3. transport 不需要为第二个 provider 重写核心逻辑
4. 至少能稳定支持一种共同能力，比如 chat completion

如果你能做到这 4 件事，你的 provider abstraction 就算真正成立了。

## 简历怎么写更像真的做过网关

- implemented a unified provider abstraction for a multi-provider AI gateway, normalizing request and response formats behind a shared execution core
- added a second provider to a Bifrost-inspired gateway without changing transport or orchestration logic
- separated OpenAI-compatible provider reuse from non-compatible provider-specific conversion flows

这三句在你完成 lesson 5 作业后都非常可信。

## 下一课
Lesson 6 应该进入 reliability features：

- retries
- backoff
- key selection
- queue behavior
- fallback continuation rules

这一节会让你的 bifrost-lite 从“能跑”进入“更像一个真正网关”。 
