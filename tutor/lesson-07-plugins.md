# Lesson 7: 插件扩展层

## 这一课学完你能做什么
你可以解释 Bifrost 为什么没有把治理、日志、缓存、观测这些能力硬写死在 core 里；你也能说清插件接口、pre/post hook 的执行顺序、short-circuit 的意义，以及自己的 `bifrost-lite` 应该先实现哪种最小插件系统。

## 这一层的职责
插件层负责给核心执行链路加“横切能力”，但不破坏 core 的主职责。

它主要承担这些事：

1. 在 provider 调用前做检查或改写
2. 在 provider 返回后做记录、恢复或修正
3. 在 HTTP transport 层拦截请求/响应
4. 在 streaming 过程中拦截 chunk
5. 在请求结束后把 trace 注入观测后端

一句话说：插件层让 Bifrost 从“有执行引擎”进化成“可扩展平台”。

## 它在系统里的位置

上游：
- HTTP transport
- core request lifecycle
- MCP tool execution

当前层：
- `HTTPTransportPlugin`
- `LLMPlugin`
- `MCPPlugin`
- `ObservabilityPlugin`
- `PluginPipeline`

下游：
- logging
- governance
- semantic cache
- telemetry / observability

所以插件层不是附属功能，而是 Bifrost 产品能力的重要挂载点。

## 关键执行链路

### 1. 插件种类不是一种，而是四种
定义在 [`core/schemas/plugin.go`](/mnt/windows/source/go/bifrost/core/schemas/plugin.go:208) 之后。

四类接口分别是：

1. `HTTPTransportPlugin`
2. `LLMPlugin`
3. `MCPPlugin`
4. `ObservabilityPlugin`

它们各自负责不同阶段：

- `HTTPTransportPlugin`：只在 HTTP transport 层生效
- `LLMPlugin`：每次 LLM 请求都能拦
- `MCPPlugin`：每次 MCP tool execution 都能拦
- `ObservabilityPlugin`：异步接收完成 trace

这说明 Bifrost 对“扩展点”是分层设计的，不是只给一个万能 hook。

### 2. 执行顺序是有明确语义的
`plugin.go` 里直接写明了执行顺序，最重要的是这条：

1. `HTTPTransportPreHook`
2. `PreLLMHook`
3. provider call
4. `PostLLMHook`
5. `HTTPTransportPostHook`
6. streaming 时改为 `HTTPTransportStreamChunkHook`

更关键的是：

- Pre hooks 按注册顺序执行
- Post hooks 按反向顺序执行

这就是经典“包裹式”语义。

你可以把它理解成：

```text
PluginA.Pre
  PluginB.Pre
    Provider
  PluginB.Post
PluginA.Post
```

这对 logging、cache、治理这种层叠能力非常重要。

### 3. `PluginPipeline` 是真正的执行器
在 core 里，插件并不是随手 `for` 一圈调用，而是统一交给 [`PluginPipeline`](/mnt/windows/source/go/bifrost/core/bifrost.go:139)。

最关键的两个方法是：

- [`RunLLMPreHooks(...)`](/mnt/windows/source/go/bifrost/core/bifrost.go:5538)
- [`RunPostLLMHooks(...)`](/mnt/windows/source/go/bifrost/core/bifrost.go:5587)

`RunLLMPreHooks(...)` 做的事情是：

1. 按顺序执行所有 `PreLLMHook`
2. 记录有多少插件真正跑过 pre-hook
3. 捕获 short-circuit
4. 即使 plugin 报错，也只记录 warning，不中断整个系统

`RunPostLLMHooks(...)` 则会：

1. 只对“已经跑过 pre-hook 的插件”执行 post-hook
2. 逆序执行
3. 允许插件恢复错误或把响应作废
4. streaming 时累计 timing，而不是每 chunk 都建一堆 span

这说明插件系统不是“能调用 hook 就行”，而是认真处理了生命周期对称性。

### 4. short-circuit 是插件系统的关键能力
定义在 [`core/schemas/plugin_native.go`](/mnt/windows/source/go/bifrost/core/schemas/plugin_native.go:19)。

`LLMPluginShortCircuit` 可以携带：

- `Response`
- `Stream`
- `Error`

这意味着插件可以在 provider 调用前直接决定：

1. 返回一个缓存命中的响应
2. 返回一个错误
3. 直接给一个流

而且一旦 short-circuit，Bifrost 不会继续打 provider，但仍然会对已经执行过 pre-hook 的插件跑对应 post-hook。

这点很强，因为它保证了：

- cache 命中可以直接返回
- governance 拒绝可以提前终止
- logging/metrics 仍然有机会补尾

### 5. plugin 错误默认不会直接暴露给用户
`plugin.go` 里明确写了：

- 插件内部 error 会被记录为 warning
- 不是所有 plugin error 都直接传给调用者

这是一种很成熟的产品思路。

因为插件本来就是“增强层”，默认不应该让整个网关因为某个附属能力崩掉。

当然，插件仍然可以通过 `LLMPluginShortCircuit.Error` 或返回 `BifrostError` 来有意改变主链路行为。

这和“插件自己挂了”是两回事。

## 代表性插件例子

### 1. semantic cache：典型的 success short-circuit
看 [`plugins/semanticcache/main.go`](/mnt/windows/source/go/bifrost/plugins/semanticcache/main.go:406) 的 `PreLLMHook(...)`。

它的逻辑很像：

1. 先看是否有 cache key
2. 判断这个 request type 是否支持缓存
3. 尝试 direct search
4. 没命中再尝试 semantic search
5. 如果命中，就返回 short-circuit

这就是插件系统最漂亮的一种用法：

- 不改 core
- 不改 handler
- 直接在 pre-hook 上插一个“可能提前成功返回”的能力

它很适合你在面试里讲“插件化缓存设计”。

`PostLLMHook(...)` 则负责在 provider 成功返回后把内容写入缓存。

这就是非常标准的：

- pre-hook 查缓存
- post-hook 回填缓存

### 2. governance：典型的 policy gate
看 [`plugins/governance/main.go`](/mnt/windows/source/go/bifrost/plugins/governance/main.go:1029) 的 `PreLLMHook(...)`。

它会：

1. 检查是否跳过 key selection
2. 校验必需 header
3. 提取 virtual key / user id / provider / model
4. 做治理评估
5. 如果不允许，直接返回 `LLMPluginShortCircuit{Error: ...}`

这说明 governance 插件本质上是“执行前策略闸门”。

而它的 `PostLLMHook(...)` 会在请求完成后更新 usage、过滤模型列表等。

这正好体现了 pre/post hook 的天然分工：

- pre：准入控制
- post：事后记账

### 3. logging：典型的前后拼接型插件
看 [`plugins/logging/main.go`](/mnt/windows/source/go/bifrost/plugins/logging/main.go:395) 和 [`PostLLMHook(...)`](/mnt/windows/source/go/bifrost/plugins/logging/main.go:577)。

它的设计很有意思：

`PreLLMHook(...)` 主要做：

1. 提取 request id
2. 把输入、参数、tools、metadata 等先抓下来
3. 对流式请求创建 stream accumulator
4. 把这些“前半段数据”放进 `pendingLogs`

`PostLLMHook(...)` 再做：

1. 根据 request id 取回 pending data
2. 加上 output、error、selected key、retry count、latency
3. 拼成完整 log entry
4. 异步写出

这是一种非常典型、也非常好的插件模式：

- pre-hook 拿输入
- post-hook 拿输出
- 两边拼起来形成完整审计日志

### 4. telemetry / observability：插件也可以只做观测，不改主结果
例如 `plugins/telemetry` 里的 `PreLLMHook/PostLLMHook` 主要是记录 metrics。

这类插件通常：

- 不 short-circuit
- 不改变请求/响应
- 只在 context 或 trace 上附加观测信息

这说明插件系统不只是为“改行为”准备的，也为“看行为”准备的。

## streaming 情况下插件怎么工作

### 1. LLM post-hook 仍然存在
streaming 请求仍会经过 `PreLLMHook` / `PostLLMHook`，但核心处理方式不太一样：

- `RunPostLLMHooks(...)` 会识别 streaming mode
- 对每个 plugin 的 timing 做累计
- 最后在 stream 结束时统一收尾 span

这是为了避免每个 chunk 都生成一堆 span，导致 trace 爆炸。

### 2. HTTP transport 还有专门的 chunk hook
`HTTPTransportPlugin` 还定义了：

- [`HTTPTransportStreamChunkHook(...)`](/mnt/windows/source/go/bifrost/core/schemas/plugin.go:254)

这个 hook 是“每个 chunk 都能拦”的。

返回语义也非常实用：

- `(*chunk, nil)`：继续发送
- `(nil, nil)`：跳过这个 chunk
- `(*chunk, error)`：告警但继续
- `(nil, error)`：直接回错误并停止流

这使得 transport 层也能做：

- chunk 级日志
- chunk 过滤
- streaming redaction

## 为什么这是项目亮点

### 1. 它把横切需求从 core 解耦出来了
如果没有插件系统，Bifrost 的 core 会很快变成一团：

- if logging enabled ...
- if governance enabled ...
- if cache enabled ...
- if metrics enabled ...

而现在这些能力大多变成了插件。

这不仅更清晰，也更适合持续扩展。

### 2. short-circuit 让插件不只是“旁观者”
很多项目说自己有 middleware，但 middleware 只能看看请求，不能真正改变主流程。

Bifrost 不一样。

通过 short-circuit，插件可以：

- 拒绝请求
- 提前返回缓存
- 恢复错误
- 替换响应

这让插件真正成为平台能力，而不只是回调函数。

### 3. pre/post 对称性设计很强
“执行过 pre-hook 的插件一定会执行对应 post-hook” 这点很重要。

因为很多插件天然需要这种配对语义：

- logging
- tracing
- governance usage update
- cache writeback

如果没有这个对称性，插件作者会很痛苦，逻辑也会很脆弱。

### 4. 插件错误默认不拖垮主链路
这是一种很成熟的平台态度：

- 插件很重要
- 但默认不能比主链路更重要

对生产系统来说，这比“所有东西都严格失败”更实用。

## Bifrost 设计 takeaway

### 1. 先做 hook system，再做具体插件
你自己做 `bifrost-lite` 时，千万不要先写“日志逻辑”“缓存逻辑”。

应该先写：

```go
type Plugin interface {
    Pre(ctx, req) (...)
    Post(ctx, resp, err) (...)
}
```

先把挂载点做出来，后面能力才能自然长进去。

### 2. 第一个值得做的插件通常不是治理，而是 logging
因为 logging 最容易验证：

- pre-hook 拿输入
- post-hook 拿输出
- 不改变主链路
- 最容易看到效果

这很适合作为 `bifrost-lite` 的第一号插件。

### 3. 第二个值得做的插件是 cache stub
哪怕不做真正 semantic cache，也建议你先做一个“命中则 short-circuit”的 cache stub。

因为这样你就能真正体会：

- short-circuit 为什么重要
- pre/post hook 为什么要对称

## 我自己这一课要实现什么

### 这节课你的 bifrost-lite 目标
做一个最小插件系统，让 logging 和一个简单 cache 插件能挂进主链路。

### 最小接口建议

```go
type Plugin interface {
    Name() string
    Pre(ctx context.Context, req *Request) (*Request, *ShortCircuit, error)
    Post(ctx context.Context, resp *Response, err error) (*Response, error, error)
}
```

你也可以更简单一点，但至少要保留：

- pre
- post
- short-circuit

### 建议先做两个插件

#### 1. `logging` 插件
功能：

- pre 记录输入
- post 记录输出、耗时、错误

#### 2. `cache` 插件
功能：

- pre 查内存 map
- 命中则 short-circuit 返回缓存
- post 把成功响应写回缓存

### 这一课的完成标准
你要做到：

1. Gateway 支持按顺序执行 pre-hooks
2. Post-hooks 按逆序执行
3. cache 命中时可以跳过 provider 调用
4. logging 能拿到完整输入输出

如果你能完成这 4 件事，你的 bifrost-lite 就开始具备“平台感”了。

## 简历怎么写更像平台工程

- built a plugin pipeline for a Bifrost-inspired AI gateway with ordered pre-hooks, reverse post-hooks, and short-circuit support
- implemented pluggable logging and cache behaviors without changing the core request execution path
- designed a middleware-style extension layer for governance, observability, and request/response interception

这类表述比“写了几个 if 开关”强很多。

## 下一课
Lesson 8 最适合挑 Bifrost 最有简历价值的高级能力：

- MCP integration
- semantic caching
- governance / virtual keys
- observability
- UI / control plane

那一节的目标不是继续拆所有代码，而是帮你挑出“最值得复刻进 bifrost-lite、也最值得写进简历”的 2 到 3 个亮点。 
