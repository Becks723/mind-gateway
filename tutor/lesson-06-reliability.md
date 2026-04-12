# Lesson 6: 可靠性能力

## 这一课学完你能做什么
你可以解释 Bifrost 为什么不是“请求失败就报错”的简单代理，而是一个带重试、退避、key 选择、队列控制、fallback 控制的网关；你也可以把这些能力拆成 `bifrost-lite` 里最值得优先补齐的可靠性模块。

## 这一层的职责
这一层要解决的是“请求失败时怎么办”和“系统压力上来时怎么办”。

主要包括 5 类能力：

1. retry
2. backoff
3. key selection
4. queue behavior
5. fallback continuation rules

这些能力叠在一起，才让 Bifrost 从“能转发请求”变成“能扛真实线上波动的网关”。

## 它在系统里的位置

上游：
- `handleRequest(...)`
- `handleStreamRequest(...)`
- `tryRequest(...)`
- `tryStreamRequest(...)`

当前层：
- `executeRequestWithRetries(...)`
- `selectKeyFromProviderForModel(...)`
- `WeightedRandomKeySelector(...)`
- `shouldTryFallbacks(...)`
- `shouldContinueWithFallbacks(...)`
- queue 满时的投递逻辑

下游：
- provider HTTP 调用
- 插件返回的错误
- key store / account config

也就是说，可靠性逻辑不是 provider 独有，也不是 transport 独有，而是 core orchestration 层统一控制的。

## 关键执行链路

### 1. retry 发生在 worker 内，不发生在 handler 内
真正的重试总控在 [`executeRequestWithRetries(...)`](/mnt/windows/source/go/bifrost/core/bifrost.go:4658)。

它的执行模式是：

1. 根据 `config.NetworkConfig.MaxRetries` 控制最大重试次数
2. 每次尝试前把 `NumberOfRetries` 写入 context
3. 如果是重试轮次，先记录重试日志
4. 按 backoff 等待一段时间
5. 再执行真实 provider request handler
6. 根据错误类型判断是否继续重试

这个位置选得非常对，因为 retry 属于执行策略，而不是 HTTP 解析逻辑。

### 2. retry 不是“所有错误都重试”
在 [`executeRequestWithRetries(...)`](/mnt/windows/source/go/bifrost/core/bifrost.go:4722) 后半段，Bifrost 只会对特定错误重试。

能触发 retry 的主要是两类：

1. 网络/请求执行错误
2. rate limit 错误

源码里能直接看到这些判断：

- `ErrProviderDoRequest`
- `ErrProviderNetworkError`
- retryable status code
- 错误 message/type/code 被识别为 rate limit

不会继续重试的典型情况包括：

- 已经成功
- `IsBifrostError == true` 的内部错误
- 请求被取消
- 普通不可重试业务错误

这很关键，因为盲重试只会把坏情况放大。

### 3. backoff 是按 provider network config 配的
重试前会读取 provider 的 network config：

- `MaxRetries`
- `RetryBackoffInitial`
- `RetryBackoffMax`

这些配置定义在 [`core/schemas/provider.go`](/mnt/windows/source/go/bifrost/core/schemas/provider.go:46)。

也就是说，重试策略不是写死在代码里，而是 provider 级别可配置。

这很像真实网关系统，因为不同 provider 的稳定性、速率限制和网络特征通常不同。

我在当前仓库里能直接确认到 `calculateBackoff(...)` 被调用，但没直接检索到它的定义位置；能确认的事实是：Bifrost 确实使用了按 provider config 计算出的 backoff，而不是立即紧贴重试。

### 4. streaming 请求也有“首块错误检测”
这是一个非常好的细节。

在 [`executeRequestWithRetries(...)`](/mnt/windows/source/go/bifrost/core/bifrost.go:4747) 附近，Bifrost 对 streaming 成功返回的 channel 还会额外检查：

- 第一块 stream chunk 是否其实是错误

原因是有些 provider 会：

- HTTP 200 成功建立 SSE
- 但第一块 event 才真正携带 rate limit / provider error

如果不做这一步，就没法对这类流式错误进行 retry/fallback。

这说明 Bifrost 对 streaming reliability 的理解是比较深入的。

## key selection 是第二个可靠性核心

### 1. 一个 provider 可以有多把 key
key selection 逻辑在 [`selectKeyFromProviderForModel(...)`](/mnt/windows/source/go/bifrost/core/bifrost.go:6128)。

这一步不是“随便拿第一把 key”，而是要做很多过滤：

1. direct key 是否已在 context 里显式指定
2. 是否允许 skip key selection
3. provider 下到底有哪些 keys
4. batch/file 请求是否只能用 `UseForBatchAPI`
5. 当前 model 是否被 allow / blacklist
6. Azure / Bedrock / Vertex / Replicate / VLLM 是否有 deployment 级支持
7. header 中是否显式指定 `key id` 或 `key name`

最后才进入真正“选一把 key”的阶段。

### 2. weighted random 是默认 key selector
默认 key selector 是 [`WeightedRandomKeySelector(...)`](/mnt/windows/source/go/bifrost/core/bifrost.go:6380)。

它的逻辑很直接：

1. 统计所有 key 的权重总和
2. 如果总权重是 0，就随机均匀选
3. 否则按权重区间随机选中一把 key

这意味着：

- 多 key 可以做负载分配
- 流量不是硬编码平均分
- provider key 可以按权重倾斜

这是一个非常典型的网关能力。

### 3. session stickiness 会覆盖“每次随机选 key”
`selectKeyFromProviderForModel(...)` 里还有一个很强的细节：session stickiness。

如果 context 里有 `session id`，且：

- `kvStore` 可用
- 当前不是 fallback 分支

它会：

1. 先尝试从 KV 里取这次 session 绑定的 key
2. 如果有，就继续用它
3. 如果没有，再选一把新的 key
4. 再写回 KV，形成会话级绑定

这很适合那些要求多轮会话尽量走同一后端 key / deployment 的场景。

### 4. fallback 时不会继续沿用 primary 的 sticky key
代码里一个很容易忽略但很重要的点是：

- stickiness 只在 `fallbackIndex == 0` 时生效

也就是说，primary provider 的 session 绑定 key，不会强行延续到 fallback provider。

这是合理的，因为 fallback 的目标是恢复可用性，而不是坚持原始 key 绑定。

## queue behavior 是第三个可靠性核心

### 1. queue 满时有两种策略
在 [`tryRequest(...)`](/mnt/windows/source/go/bifrost/core/bifrost.go:4346) 和对应 streaming 路径中，Bifrost 对 queue 满的处理不是固定的。

它看 `dropExcessRequests`：

- `false`：阻塞等待 queue 空位
- `true`：直接丢弃请求并返回错误

这两个策略没有谁绝对更好，而是适合不同场景：

- 等待：更保成功率，但会增加延迟
- 丢弃：更保系统稳定，但会牺牲部分请求

这已经是很像真实网关的背压策略了。

### 2. queue 投递前还会检查 provider 是否正在关闭
请求入队前，Bifrost 不只是尝试发送到 channel，还会检查：

- `pq.isClosing()`
- `pq.done`
- `ctx.Done()`

这意味着在 provider 热更新、移除或 shutdown 的时候：

- 不会再继续盲目往关闭中的 queue 里送消息
- 也能避免 `send on closed channel`

这是 Lesson 3 里 `ProviderQueue` 设计的实际价值体现。

### 3. queue 满不是 panic，而是显式错误
如果开启 `dropExcessRequests`，Bifrost 会明确返回：

- `"request dropped: queue is full"`

这非常重要，因为生产系统最怕的不是失败，而是：

- 无声失败
- 行为不可解释

Bifrost 在这里至少把失败显式建模成了一个可观察的错误。

## fallback 规则是第四个可靠性核心

### 1. primary 失败不一定就 fallback
在 [`shouldTryFallbacks(...)`](/mnt/windows/source/go/bifrost/core/bifrost.go:3884) 里，Bifrost 会先判断要不要继续尝试 fallback。

如果出现以下情况，就不会 fallback：

1. 根本没有 primary error
2. 请求被取消
3. `AllowFallbacks == false`
4. 请求没有配置 fallback provider

这说明 fallback 不是“无脑试下一个”，而是受语义控制的。

### 2. fallback attempt 之间也有停止规则
在 [`shouldContinueWithFallbacks(...)`](/mnt/windows/source/go/bifrost/core/bifrost.go:3999) 里，Bifrost 还会判断：

- 某个 fallback 失败后，要不要继续试下一个 fallback

会停止继续 fallback 的典型情况：

1. 请求被取消
2. `AllowFallbacks == false`

否则才继续往后试。

这意味着 plugin 或 provider 返回的错误，可以显式地告诉 core：

- “这个错误允许 fallback”
- “这个错误不要再 fallback 了，立刻结束”

### 3. `AllowFallbacks` 是插件和 core 之间的重要协议
在 [`core/schemas/bifrost.go`](/mnt/windows/source/go/bifrost/core/schemas/bifrost.go:868) 里，`BifrostError` 的 `AllowFallbacks` 语义非常清楚：

- `nil`：默认允许 fallback
- `true`：允许 fallback
- `false`：禁止 fallback

这让 plugin 可以非常精细地控制系统行为。

例如：

- rate limit plugin 可能允许 fallback
- 某些鉴权/治理拒绝可能明确禁止 fallback

这就是平台系统里非常有价值的“错误语义化”。

## Bifrost 设计 takeaway

### 1. reliability 不是一个点，而是一组配合工作的机制
在 Bifrost 里，可靠性不是只有 retry。

它至少是下面这些东西一起工作：

- retry
- backoff
- weighted key selection
- queue 背压
- fallback
- error semantics

只有把这些组合起来，你的网关才会像网关，而不是 SDK wrapper。

### 2. 重试与 fallback 不是一回事
这点在面试里很值得讲清楚。

- retry：同一个 provider attempt 内，先再试几次
- fallback：当前 provider 失败后，换下一个 provider/model

这两个策略分别解决不同问题：

- retry 解决瞬时波动
- fallback 解决 provider 级失败或不适配

### 3. key selection 其实也是可靠性的一部分
很多人把 key selection 只看成“负载均衡”。

但在网关里，它同时也是可靠性问题，因为：

- 单 key 被限流时，多 key 能分流
- session stickiness 能降低会话抖动
- model/deployment 过滤能避免错误路由

## 我自己这一课要实现什么

### 这节课你的 bifrost-lite 目标
给已经能跑通的主链路加上第一批“抗波动能力”。

### 推荐你按这个顺序做

#### 1. 先做 retry + backoff
最小版本就够：

- `maxRetries`
- `initialBackoff`
- `maxBackoff`
- 只对网络错误和 429 重试

#### 2. 再做 fallback
最小版本：

- request 里允许传 `fallbacks`
- primary 失败后尝试下一个 provider/model

#### 3. 再做多 key 选择
最小版本：

- 每个 provider 配多把 key
- 按权重随机选一把

#### 4. 最后做 queue 策略
最小版本：

- queue 满时阻塞
- 或者直接返回 `queue full`

### 这一课的最小实现建议

```text
internal/core/retry.go
internal/core/fallback.go
internal/core/keyselect.go
internal/core/queue.go
```

### 这一课的完成标准
你要做到：

1. provider 临时 429 或网络错误时会自动 retry
2. primary provider 挂掉时能 fallback 到第二个 provider
3. 一个 provider 配多 key 时不会永远打同一把
4. queue 满时系统行为是显式可控的

如果你做到这 4 件事，你的 bifrost-lite 就从“学习项目”开始进入“像一个小型生产网关”。

## 简历怎么写更像真实基础设施工作

- implemented reliability features for a Bifrost-inspired AI gateway, including retries, backoff, fallback routing, and weighted key selection
- added queue backpressure handling and explicit request-drop behavior for overload protection
- designed error semantics that control whether failed requests should continue to fallback providers

这些说法的前提是你确实把这几个机制做出来，而不是只停留在文档层。

## 下一课
Lesson 7 最适合进入 extensibility via plugins：

- pre-hooks 和 post-hooks 怎么跑
- short-circuit 怎么工作
- logging / governance / semantic cache 这种能力为什么适合做成插件

这一节会让你理解 Bifrost 为什么不是“功能都写死在 core 里”。 
