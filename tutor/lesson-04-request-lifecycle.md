# Lesson 4: 一条请求如何跑完整个系统

## 这一课学完你能做什么
你可以把一次 `/v1/chat/completions` 请求从 HTTP 入口一路讲到 provider 调用、fallback、插件、最终响应回写；同时你也能区分 Bifrost 的非流式和流式两条执行路径。

## 这一层的职责
这一课不是讲某个模块本身，而是讲“模块之间如何接力”。

你要理解的是：

1. handler 如何把 HTTP 请求变成内部请求
2. context 如何把 header、request id、治理信息带进核心层
3. core 如何执行主 provider 与 fallback
4. plugin hooks 在链路中插在什么位置
5. 响应最后如何回到 HTTP 层

如果 Lesson 3 讲的是“发动机结构”，那 Lesson 4 讲的是“发动机怎么把车开起来”。

## 先看整条链路

### 非流式 `/v1/chat/completions`
1. 路由命中 [`CompletionHandler.RegisterRoutes(...)`](/mnt/windows/source/go/bifrost/transports/bifrost-http/handlers/inference.go:593)
2. 进入 [`chatCompletion(...)`](/mnt/windows/source/go/bifrost/transports/bifrost-http/handlers/inference.go:914)
3. `prepareChatCompletionRequest(...)` 把 JSON body 转为 `BifrostChatRequest`
4. `ConvertToBifrostContext(...)` 把 HTTP 上下文转为 `BifrostContext`
5. 调 `h.client.ChatCompletionRequest(...)`
6. 进入 [`makeChatCompletionRequest(...)`](/mnt/windows/source/go/bifrost/core/bifrost.go:646)
7. 包成 `BifrostRequest`，进入 [`handleRequest(...)`](/mnt/windows/source/go/bifrost/core/bifrost.go:4017)
8. `handleRequest()` 先试 primary provider，不行再试 fallbacks
9. `tryRequest()` 跑 pre-hooks、入队、等 worker 回结果、跑 post-hooks
10. `requestWorker()` 选 key、retry、调用 provider
11. provider 返回统一 `BifrostResponse`
12. handler 调 `SendJSON(...)` 回给客户端

### 流式 `/v1/chat/completions`
1. 前面步骤基本一样
2. 如果 `stream=true`，则转入 [`handleStreamingChatCompletion(...)`](/mnt/windows/source/go/bifrost/transports/bifrost-http/handlers/inference.go:1457)
3. 调 `ChatCompletionStreamRequest(...)`
4. 进入 [`handleStreamRequest(...)`](/mnt/windows/source/go/bifrost/core/bifrost.go:4141)
5. `tryStreamRequest()` 入队并等 `ResponseStream`
6. worker 拿到 stream channel 后写回
7. handler 把每个 chunk 包装成 SSE event 输出
8. 流结束时发送 `[DONE]` 或直接关闭 stream

这就是 Bifrost 端到端请求的核心骨架。

## 关键执行链路

### 1. handler 先做输入归一化，不直接做 provider 调用
对于 chat 请求，真正的入口是 [`chatCompletion(...)`](/mnt/windows/source/go/bifrost/transports/bifrost-http/handlers/inference.go:914)。

它先调用 [`prepareChatCompletionRequest(...)`](/mnt/windows/source/go/bifrost/transports/bifrost-http/handlers/inference.go:845)：

1. 用 `sonic.Unmarshal` 解析原始 JSON
2. 通过 `schemas.ParseModelString(req.Model, "")` 解析出 `provider/model`
3. 解析 `fallbacks`
4. 校验 `messages` 非空
5. 提取 `extra_params`
6. 组装成 `schemas.BifrostChatRequest`

你要注意这个设计点：

- transport 层负责 HTTP/body 语义
- core 层只接统一内部结构

这让 core 不需要关心 HTTP body 是怎么长的。

### 2. `ConvertToBifrostContext()` 把 HTTP 世界搬进内部世界
在进入 core 之前，handler 会调用 [`ConvertToBifrostContext(...)`](/mnt/windows/source/go/bifrost/transports/bifrost-http/lib/ctx.go:95)。

它做的事非常关键：

1. 创建或复用 request-scoped `BifrostContext`
2. 设置 `request_id`
3. 把 user values 从 `fasthttp.RequestCtx` 拷进来
4. 处理 `x-bf-prom-*`
5. 处理 `x-bf-maxim-*`
6. 处理 `x-bf-mcp-*`
7. 处理治理相关 header，比如 `x-bf-vk`
8. 处理显式 key 选择，比如 `x-bf-api-key` / `x-bf-api-key-id`
9. 处理额外透传 header、session stickiness 等

这一步决定了 Bifrost 后面的执行链路不只是拿到“一个请求体”，而是拿到带完整上下文的运行时请求。

对网关来说，这很重要，因为真实请求里常常还要带：

- 跟踪信息
- 治理信息
- 选 key 信息
- 过滤信息
- cache/session 信息

### 3. core 入口先包成统一请求对象
`chatCompletion()` 最终会调到 [`ChatCompletionRequest(...)`](/mnt/windows/source/go/bifrost/core/bifrost.go:685)。

这里有个很值得你模仿的设计：

- 公开 API 方法先尽量薄
- 真正共用逻辑沉到 `makeChatCompletionRequest(...)`

`makeChatCompletionRequest(...)` 做的核心事情是：

1. 校验请求
2. 从对象池取一个 `BifrostRequest`
3. 设置 `RequestType = ChatCompletionRequest`
4. 把 `ChatRequest` 塞进去
5. 调 `handleRequest(...)`

这意味着所有上层入口最后都会收敛成一套统一执行格式。

### 4. `handleRequest()` 是非流式总控
非流式请求真正的大总控在 [`handleRequest(...)`](/mnt/windows/source/go/bifrost/core/bifrost.go:4017)。

它按顺序做：

1. `defer releaseBifrostRequest(req)`
2. `validateRequest(req)`
3. 确保 context 不为 nil
4. 设置 `FallbackIndex=0`
5. 确保 `request_id` 存在
6. `tryRequest(ctx, req)` 先跑主 provider
7. 如果主 provider 失败，判断要不要进 fallback
8. 如果要 fallback，就循环尝试每个 fallback provider/model
9. 每次 fallback 前都会 `prepareFallbackRequest(...)`
10. 如果某个 fallback 成功，立即返回
11. 全部失败则返回 primary error

这让你看到一个重要事实：

Bifrost 的 fallback 不是 transport 层做的，也不是 provider 自己做的，而是 core orchestration 层统一做的。

### 5. fallback 是通过“复制请求并替换 provider/model”实现的
fallback 构造逻辑在 [`prepareFallbackRequest(...)`](/mnt/windows/source/go/bifrost/core/bifrost.go:3917)。

它不是重新构造整条请求，而是：

1. 复制原始 `BifrostRequest`
2. 针对不同 request type 拷贝对应子请求
3. 只改 `Provider` 和 `Model`

这个做法很实用，因为：

- 原始输入不会丢
- fallback 逻辑对不同 request type 可以复用
- 只替换 provider/model，风险最小

### 6. `tryRequest()` 才是主 provider 的真正执行器
真正把请求送进 worker 模型的是 [`tryRequest(...)`](/mnt/windows/source/go/bifrost/core/bifrost.go:4251)。

关键步骤如下：

1. `getProviderQueue(provider)`
2. 如果 MCP 开启，就先注入 tools
3. 取 tracer 并写入 context
4. 从 pool 中拿 `PluginPipeline`
5. 执行 `RunLLMPreHooks(...)`
6. 处理可能的 short-circuit
7. 从 pool 中获取 `ChannelMessage`
8. 把消息发进 `pq.queue`
9. 阻塞等待 `msg.Response` 或 `msg.Err`
10. 拿到结果后执行 `RunPostLLMHooks(...)`
11. 释放 `ChannelMessage`

也就是说，主 provider 的成功路径并不是：

```text
handler -> provider
```

而是：

```text
handler -> core -> pre-hooks -> queue -> worker -> provider -> post-hooks -> handler
```

这才是 Bifrost 最值得复刻的链路。

### 7. worker 负责真正的 provider 调用、key selection 和 retry
进入 worker 后，请求会落到 [`requestWorker(...)`](/mnt/windows/source/go/bifrost/core/bifrost.go:4863)。

worker 做的事情包括：

1. 从 queue 中取消息
2. 判断 provider 是否需要 key
3. 获取单 key 或多 key
4. 设置 context 里的 selected key 信息
5. 对 streaming 请求准备 `postHookRunner`
6. 调 `executeRequestWithRetries(...)`
7. 真正进入 `handleProviderRequest(...)`
8. 最后把结果写回 `msg.Response` 或 `msg.Err`

这说明 retry 与 key selection 都是在 worker 层完成，而不是在 handler 层。

这是对的，因为它们属于执行策略，不属于 HTTP 语义。

### 8. `handleProviderRequest()` 只是统一分发，不再管 HTTP
在 [`handleProviderRequest(...)`](/mnt/windows/source/go/bifrost/core/bifrost.go:5041) 里，Bifrost 根据 `RequestType` 调用 provider 的具体方法。

对于本课关注的 chat 路径，关键分支是：

- `schemas.ChatCompletionRequest`
- `provider.ChatCompletion(req.Context, key, req.BifrostRequest.ChatRequest)`

这一层的价值是把 provider 接口统一调度掉，而不是做协议转换。

协议转换早在 transport 层就结束了。

### 9. handler 最后只做 HTTP 回写
当 core 返回结果后，`chatCompletion()` 会做几件很干净的事：

1. 如果是错误，就 `SendBifrostError(...)`
2. 如果 provider 带了 response headers，就透传
3. 如果启用了 large response mode，就直接流大响应体
4. 否则 `SendJSON(...)`

可以看 [`SendJSON(...)`](/mnt/windows/source/go/bifrost/transports/bifrost-http/handlers/utils.go:24) 和 [`SendBifrostError(...)`](/mnt/windows/source/go/bifrost/transports/bifrost-http/handlers/utils.go:55)。

这一步很像“最后一公里适配器”。

这也说明一个成熟架构特征：handler 的后半段几乎只做 transport serialization。

## 流式请求怎么不一样

### 1. 入口处分叉
在 [`chatCompletion(...)`](/mnt/windows/source/go/bifrost/transports/bifrost-http/handlers/inference.go:926) 里，如果 `req.Stream != nil && *req.Stream`，就转入 [`handleStreamingChatCompletion(...)`](/mnt/windows/source/go/bifrost/transports/bifrost-http/handlers/inference.go:1457)。

这说明 stream/non-stream 在 transport 层就分流了。

### 2. 统一 streaming handler
`handleStreamingChatCompletion()` 本身非常薄，只是构造：

```go
getStream := func() (chan *schemas.BifrostStreamChunk, *schemas.BifrostError) {
    return h.client.ChatCompletionStreamRequest(bifrostCtx, req)
}
```

然后交给 [`handleStreamingResponse(...)`](/mnt/windows/source/go/bifrost/transports/bifrost-http/handlers/inference.go:1506)。

这个设计很好，因为：

- chat stream
- responses stream
- speech stream
- transcription stream

都能共用一套 HTTP streaming 输出逻辑。

### 3. `handleStreamRequest()` 是 streaming 总控
core 侧对应的是 [`handleStreamRequest(...)`](/mnt/windows/source/go/bifrost/core/bifrost.go:4141)。

它与非流式 `handleRequest()` 逻辑几乎平行：

1. 校验请求
2. 先试 primary stream provider
3. 如果失败，判断要不要 fallback
4. 按顺序尝试 fallback stream provider
5. 返回 `chan *BifrostStreamChunk`

这说明 Bifrost 没有把 streaming 当作“附加输出格式”，而是有完整独立的执行控制面。

### 4. SSE 回写细节
在 [`handleStreamingResponse(...)`](/mnt/windows/source/go/bifrost/transports/bifrost-http/handlers/inference.go:1506) 里，HTTP 层会：

1. 先 `getStream()`
2. 若失败，直接返回标准 JSON 错误
3. 若成功，再设置 SSE headers
4. 创建 `SSEStreamReader`
5. 启 goroutine 逐个处理 stream chunk
6. 对每个 chunk 做可选 interceptor
7. marshal 为 JSON
8. 用 `reader.SendEvent(...)` 输出
9. 非 responses/image-gen 流结束时发送 `[DONE]`
10. 客户端断开时调用 `cancel()`

这里很值得你记住的一点是：

它先拿到 stream，再设置 SSE headers。这样 provider 初始化失败时，客户端仍能收到正常的 JSON error，而不是半截 SSE。

这是一个很细但很成熟的实现细节。

## 为什么这是项目亮点

### 1. 端到端链路非常干净
Bifrost 最强的一点不是某个函数写得多炫，而是整条链路分工清楚：

- handler：解析输入、回写输出
- context converter：搬运请求元数据
- core：路由、fallback、plugin、queue、worker
- provider：真正调厂商 API

这是一种非常适合扩展的结构。

### 2. fallback 被做成了 core 能力
很多项目的 fallback 都是 transport 层用 `if err != nil { try next }` 手搓的。

Bifrost 不是，它把 fallback 放在统一请求执行器里。

好处是：

- 非流式和流式都能复用
- 不同 request type 都能复用
- plugin/tracing/context 都保持一致

### 3. plugin 插入点非常合理
在 `tryRequest()` 里，plugin 的位置是：

- 入队前：pre-hook
- worker 返回后：post-hook

这个位置非常好，因为插件看到的是统一内部请求/响应，而不是某家 provider 私有格式。

这正是平台型系统的做法。

### 4. streaming 实现有完整收尾逻辑
SSE 往往最难处理的是：

- 什么时候设置 header
- 客户端断开怎么办
- chunk 怎么拦截
- trace 什么时候结束

Bifrost 明显把这些问题都系统考虑过了。

## Bifrost 设计 takeaway

### 1. 端到端链路应该分层，而不是一把梭
你做 `bifrost-lite` 时，应该避免：

```text
HTTP parse -> provider SDK -> write response
```

更好的结构是：

```text
HTTP parse -> internal request -> core execute -> provider -> transport writeback
```

这样你以后加 fallback、trace、plugins 时不会推倒重来。

### 2. context 是网关里很重要的基础设施
在普通应用里，context 常常只是 timeout/cancel。

但在网关里，context 还可以承载：

- request id
- tracing
- selected key
- fallback index
- governance 信息
- extra headers

你后面自己实现时，至少先把 `request_id`、`selected_key`、`fallback_index` 放进去。

### 3. stream 与 non-stream 最好早点分流
不要到快输出的时候才判断是不是 stream。

更好的做法就是像 Bifrost 这样：

- transport 层先分流
- core 层有独立 stream 入口
- worker 层有独立 stream 返回通道

## 我自己这一课要实现什么

### 这节课你的 bifrost-lite 目标
亲手做出一个最小 `/v1/chat/completions` 端到端链路。

### 你应该实现的最小版本

#### 1. HTTP handler
负责：
- 解析 JSON body
- 验证 `model`
- 拆出 `provider/model`
- 构造内部 `ChatRequest`

#### 2. Internal request object
定义一个统一内部请求：

```go
type ChatRequest struct {
    Provider string
    Model    string
    Messages []Message
    Stream   bool
}
```

#### 3. Core execute method
定义：

```go
func (g *Gateway) ChatCompletion(ctx context.Context, req *ChatRequest) (*ChatResponse, error)
```

里面先不用做插件和 fallback，先能把请求送进 provider queue。

#### 4. Streaming path
再做：

```go
func (g *Gateway) ChatCompletionStream(ctx context.Context, req *ChatRequest) (<-chan *StreamChunk, error)
```

HTTP 层用 SSE 把 chunk 按顺序写出去。

### 这一课作业的完成标准
你要做到：

1. `POST /v1/chat/completions` 能跑通
2. `model` 必须是 `provider/model`
3. 非流式请求能返回 JSON
4. `stream=true` 时能返回 SSE
5. 核心执行仍然经过 `Gateway`，不是 handler 直调 provider

如果你能做到这 5 件事，你就已经做出一个非常像样的 bifrost-lite 主链路了。

## 简历怎么写更像真实工程

- implemented an end-to-end OpenAI-compatible chat completion path with internal request normalization and provider routing
- built both synchronous JSON responses and SSE streaming responses over a unified gateway core
- reproduced a Bifrost-inspired request lifecycle with context propagation, provider fallback hooks, and transport/core separation

这些表述建立在你真正做出 lesson 4 作业之后是完全成立的。

## 下一课
Lesson 5 最适合进入 provider abstraction：

- `core/schemas/provider.go` 的接口为什么这么大
- OpenAI-compatible provider 为什么能复用一套逻辑
- 非兼容 provider 为什么要自己做 converter
- 你自己的 bifrost-lite 第二个 provider 该怎么加
