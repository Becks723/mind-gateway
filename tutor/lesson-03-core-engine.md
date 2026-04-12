# Lesson 3: 核心执行引擎

## 这一课学完你能做什么
你可以解释 Bifrost 为什么不是“handler 里直接调 provider SDK”，而是要先进入一个统一执行引擎；你也可以自己动手做出 `bifrost-lite` 的第一版核心对象，包含 provider 接口、queue、worker 和基本对象池。

## 这一层的职责
`core/bifrost.go` 这一层负责的不是某个具体 API，而是整个网关的统一调度与执行。

它主要承担 6 类职责：

1. 持有统一的 `Bifrost` 主对象
2. 管理 provider 生命周期
3. 为每个 provider 建立独立 queue 和 worker
4. 执行 plugin pipeline
5. 做 key selection、重试、fallback、stream/non-stream 分流
6. 通过对象池压低分配与 GC 开销

一句话说，这一层是 Bifrost 的“调度内核”。

## 它在系统里的位置

上游：
- HTTP handlers
- integration routers
- Go SDK 调用
- MCP agent/tool execution

当前层：
- `Bifrost` 主对象
- `ProviderQueue`
- `requestWorker`
- `tryRequest` / `tryStreamRequest`

下游：
- 各 provider 实现
- 插件系统
- tracer
- MCP manager

也就是说，transport 层把请求送进来之后，并不会直接打到 OpenAI/Anthropic，而是先进入这一层统一编排。

## 关键执行链路

### 链路 A：`Init()` 如何把核心引擎启动起来
入口在 [`core/bifrost.go`](/mnt/windows/source/go/bifrost/core/bifrost.go:178) 的 `Init()`。

它按这个顺序初始化核心对象：

1. 校验 `Account` 是否存在
2. 设置 logger
3. 初始化 tracer
4. 创建 `BifrostContext`
5. 构造 `Bifrost` 主对象
6. 设置 LLM/MCP plugin 指针
7. 初始化 `providers` 原子指针
8. 初始化多个 `sync.Pool`
9. 按 `InitialPoolSize` 预热对象池
10. 通过 `account.GetConfiguredProviders()` 拿到已配置 provider
11. 如果启用 MCP，则初始化 `MCPManager`
12. 遍历 provider，逐个 `prepareProvider(...)`

这一步说明一个关键事实：

`Bifrost` 不是“某次请求时临时 new 出来”的对象，而是常驻进程内的统一执行内核。

### 链路 B：每个 provider 都有自己独立的 queue 和 worker
provider 初始化的关键在 [`prepareProvider(...)`](/mnt/windows/source/go/bifrost/core/bifrost.go:3628)。

它做的事情很集中：

1. 为该 provider 创建一个 `ProviderQueue`
2. queue 的长度来自 `config.ConcurrencyAndBufferSize.BufferSize`
3. 创建 provider 实例
4. 把 provider 追加到 `providers` 原子切片
5. 为这个 provider 启动指定数量的 worker goroutine

也就是说，Bifrost 的并发模型不是“所有请求塞进一个全局池”，而是：

- 每个 provider 一个独立队列
- 每个 provider 一组独立 worker

这是一个非常重要的设计亮点，因为 provider 之间天然隔离。

### 链路 C：请求如何进入 queue
非流式请求的主入口逻辑在 [`tryRequest(...)`](/mnt/windows/source/go/bifrost/core/bifrost.go:4251)。

顺序是：

1. 从请求里拿到 `provider` 和 `model`
2. `getProviderQueue(provider)` 获取或运行时创建 queue
3. 如果启用了 MCP，先往 request 注入 tools
4. 准备 tracer，并塞回 context
5. 从 pool 里拿一个 `PluginPipeline`
6. 跑 `RunLLMPreHooks(...)`
7. 如果插件 short-circuit，就直接返回响应或错误
8. 从 pool 里构造 `ChannelMessage`
9. 把请求塞进 `pq.queue`
10. 等待 worker 在 `msg.Response` 或 `msg.Err` 上回传结果
11. 跑 `RunPostLLMHooks(...)`
12. 释放 `ChannelMessage` 回池

这条链路你要牢牢记住，因为它就是 Bifrost 的核心执行路径。

### 链路 D：worker 真正调用 provider
执行端在 [`requestWorker(...)`](/mnt/windows/source/go/bifrost/core/bifrost.go:4863)。

每个 worker 循环做这些事：

1. 从 provider 专属 queue 里取出 `ChannelMessage`
2. 根据 provider 类型判断是否需要 key
3. 做 key selection 或多 key 收集
4. 为 streaming 请求准备 `postHookRunner`
5. 调 `executeRequestWithRetries(...)`
6. 最终调用 `handleProviderRequest(...)` 或 `handleProviderStreamRequest(...)`
7. 将结果送回 `msg.Response` / `msg.ResponseStream` / `msg.Err`

你会发现 Bifrost 的 handler 并不直接 `provider.ChatCompletion(...)`，而是通过 queue/worker 间接执行。

这就是一个真正调度内核和普通 API wrapper 的分界线。

### 链路 E：`handleProviderRequest()` 统一分发不同请求类型
在 [`handleProviderRequest(...)`](/mnt/windows/source/go/bifrost/core/bifrost.go:5073) 里，Bifrost 根据 `RequestType` 把调用分派给对应 provider 方法：

- `ListModels`
- `TextCompletion`
- `ChatCompletion`
- `Responses`
- `Embedding`
- `Speech`
- `ImageGeneration`
- 以及 batch / file / container 等

这说明统一抽象不是一句口号，而是落实为一个大而清晰的调度分发器。

### 链路 F：streaming 与 non-streaming 是两条并行路径
流式请求走 [`tryStreamRequest(...)`](/mnt/windows/source/go/bifrost/core/bifrost.go:4370)。

和非流式请求的区别主要在：

1. 使用 `ResponseStream` channel
2. streaming 时 plugin pipeline 生命周期更长
3. post-hook span finalizer 要等流结束后再释放
4. context 里会额外放 traceID / tracer / stream 相关状态

这也能看出 Bifrost 不是“普通响应改成 SSE 输出”那么简单，而是专门把 streaming 当成一等公民。

## 关键 struct / interface / function

### `Bifrost`
位置：[`core/bifrost.go`](/mnt/windows/source/go/bifrost/core/bifrost.go:62)

这是整个内核的中心。

它持有：
- providers
- requestQueues
- waitGroups
- plugin pointers
- 各种 pools
- tracer
- MCPManager
- keySelector

你可以把它理解成“网关 runtime + scheduler + provider registry”的合体。

### `ProviderQueue`
位置：[`core/bifrost.go`](/mnt/windows/source/go/bifrost/core/bifrost.go:92)

这个结构很小，但价值很高。

它除了 `queue chan *ChannelMessage` 之外，还有：
- `done`
- `closing`
- `signalOnce`
- `closeOnce`

这套设计是为了防止 provider 热更新或移除时出现 `send on closed channel`。

这是很典型的生产级并发细节。

### `ChannelMessage`
位置：[`core/bifrost.go`](/mnt/windows/source/go/bifrost/core/bifrost.go:50)

它是 transport/core/worker 之间传输请求的统一载体。

里面同时带：
- `BifrostRequest`
- `Context`
- `Response`
- `ResponseStream`
- `Err`

这比简单传一个 request struct 更适合 worker 异步返回结果。

### `prepareProvider`
位置：[`core/bifrost.go`](/mnt/windows/source/go/bifrost/core/bifrost.go:3628)

作用：
- 建 queue
- 建 provider
- 启 worker

这是 provider 生命周期真正开始的地方。

### `getProviderQueue`
位置：[`core/bifrost.go`](/mnt/windows/source/go/bifrost/core/bifrost.go:3679)

作用：
- 优先读已有 queue
- 如果没有，则运行时创建 provider 和 queue

这意味着 provider 并不一定都得在系统启动时静态固定好，也支持运行期懒创建。

### `requestWorker`
位置：[`core/bifrost.go`](/mnt/windows/source/go/bifrost/core/bifrost.go:4863)

作用：
- 真正执行 provider 调用
- 做 key selection
- 做 retry
- 回传结果

### `WeightedRandomKeySelector`
位置：[`core/bifrost.go`](/mnt/windows/source/go/bifrost/core/bifrost.go:6380)

作用：
- 基于 key weight 选 key

虽然代码不长，但很有网关味道，因为这代表它要解决多 key 负载与配额分散问题。

### `Shutdown`
位置：[`core/bifrost.go`](/mnt/windows/source/go/bifrost/core/bifrost.go:6406)

作用：
- cancel context
- 关闭所有 provider queue
- 等待所有 worker 退出
- 清理 MCP manager
- 停 tracer
- 调 plugin cleanup

这让核心引擎不是“只会启动”，也“会收尾”。

## 为什么这是项目亮点

### 1. 每个 provider 独立 queue/worker，是非常好的网关设计
这是 Bifrost 非常值得你记住的一点。

好处很直接：

- 一个 provider 出问题，不会直接拖垮所有 provider
- 并发度可以按 provider 配置
- buffer size 可以按 provider 配置
- 热更新/移除 provider 时边界清晰

这就是“provider isolation”在代码里的真实落地。

### 2. `atomic.Pointer + sync.Map + channel` 的组合很有工程味
Bifrost 没有把所有状态塞进一把大锁里。

它把不同问题拆开：

- provider 列表：`atomic.Pointer`
- requestQueues / waitGroups：`sync.Map`
- 请求执行：`chan`

这说明它在追求并发安全的同时，也在控制锁竞争和状态可替换性。

### 3. pool 用得非常系统化
从 [`Init()`](/mnt/windows/source/go/bifrost/core/bifrost.go:178) 到 [`getChannelMessage()`](/mnt/windows/source/go/bifrost/core/bifrost.go:5868)，Bifrost 为这些对象都做了 pool：

- `ChannelMessage`
- response channel
- error channel
- response stream channel
- `PluginPipeline`
- `BifrostRequest`
- `BifrostMCPRequest`

这不是“写着玩”的优化，而是明确针对高频请求路径减少分配。

这很适合简历里强调“performance-conscious gateway design”。

### 4. queue send 的边界处理很扎实
`tryRequest()` 在发消息入队时，不是简单 `pq.queue <- msg`，而是考虑了：

- provider 是否正在 closing
- `pq.done` 是否已关闭
- `ctx.Done()`
- queue 满了怎么办
- `dropExcessRequests` 是否开启

这说明作者非常清楚，生产里的麻烦通常出在边界条件，而不是 happy path。

### 5. streaming 被当成一等执行路径
很多项目是先做同步请求，再勉强给 SSE 打个补丁。

Bifrost 明显不是，它从：

- `ResponseStream`
- streaming plugin pipeline
- tracer/span finalizer
- stream request/response handling

这些点上都体现出 streaming 是独立设计过的。

## Bifrost 设计 takeaway

### 1. handler 不应该直接调用 provider
如果你想做可扩展网关，就不要把逻辑写成：

```go
func chatHandler(...) {
  provider.ChatCompletion(...)
}
```

更好的结构是：

```text
handler -> core request object -> queue -> worker -> provider
```

这样你后面才有地方插：

- plugin
- tracing
- retry
- fallback
- rate limit
- queue control

### 2. provider 抽象之上，还需要 provider runtime
很多人以为只要有 `Provider interface` 就够了。

其实不够。

你还需要：

- provider registry
- queue
- worker lifecycle
- key selection
- shutdown

这些“运行时能力”才是网关真正的难点。

### 3. 对象池值得用在网关热路径
如果你是在做高频请求系统，请求消息体、response channel、pipeline 这类对象确实值得池化。

当然，前提是你能严格 reset，否则会把脏状态带给下一次请求。

Bifrost 明显是在用性能换复杂度，而且是有意识地这么做。

## 我自己这一课要实现什么

### 这节课你的 bifrost-lite 目标
做一个最小核心执行引擎，哪怕只支持一个 provider，也要把结构搭对。

### 你应该实现的最小版本

#### 1. `Gateway` 主对象
至少包含：
- provider registry
- request queue
- worker count
- logger

#### 2. `Provider` 接口
先只保留一个方法：

```go
type Provider interface {
    ChatCompletion(ctx context.Context, req *ChatRequest) (*ChatResponse, error)
    Name() string
}
```

#### 3. `RequestMessage`
先做成：

```go
type RequestMessage struct {
    Ctx      context.Context
    Request  *ChatRequest
    Response chan *ChatResponse
    Err      chan error
}
```

#### 4. 一个 provider 一个 queue
哪怕你现在只有 OpenAI，一个 provider 对应一个 queue 的设计也先保留。

#### 5. 一个 worker loop
worker 从 queue 拉请求，调用 provider，然后把结果写回 channel。

### 这一课作业的建议目录

```text
internal/core/gateway.go
internal/core/worker.go
internal/core/types.go
internal/providers/openai/openai.go
```

### 这一课的完成标准
你要能做到：

1. handler 不直接调 provider
2. handler 把请求交给 `Gateway`
3. `Gateway` 把请求投递到 provider queue
4. worker 消费 queue 并调用 provider
5. 请求可以正常返回
6. 进程退出时 worker 能停掉

如果你能完成这 6 件事，你的 `bifrost-lite` 就开始真正像一个 gateway 了。

## 简历怎么写更像工程而不是 demo

- implemented a Bifrost-inspired gateway core with per-provider queues and worker-based request execution
- designed a unified orchestration layer separating HTTP transport from provider invocation
- built a lightweight concurrency model for LLM routing using request channels, workers, and provider isolation

这三句都很适合在你真的做出 lesson 3 作业后使用。

## 下一课
Lesson 4 应该开始走一条真实请求链路，从 `/v1/chat/completions` 一路追到 provider 响应返回：

- handler 如何解析请求
- `BifrostRequest` 如何构造
- 非流式和流式响应怎么回到 transport 层
- fallback 和 post-hook 在链路中的具体位置
