# Lesson 1: 整体架构与产品形态

## 这一课学完你能做什么
你可以用 1 分钟讲清楚 `maximhq/bifrost` 是什么、它为什么不是一个普通 API 代理、系统的顶层模块如何分工，以及你的 `bifrost-lite` 第一版应该长什么样。

## 先说结论
Bifrost 本质上是一个“多提供商 AI Gateway + OpenAI 兼容层 + 可插拔能力层 + MCP 工具网关 + 运维控制面”。

它不是只把请求转发给 OpenAI，而是把：

1. 多家模型厂商的差异接口
2. 统一请求/响应模型
3. 路由、降级、重试、负载分配
4. 插件化治理、缓存、日志、观测
5. Web UI 和配置存储

折叠成一个统一产品。

## 这一层的职责

### 1. `transports/`：对外入口层
这一层负责“接住外部世界”。

- 暴露 OpenAI 风格 API，比如 `/v1/chat/completions`
- 暴露兼容路由，比如 `/openai/...`、`/anthropic/...`、`/genai/...`
- 解析 HTTP 请求、做中间件链、把请求转成 Bifrost 内部格式
- 把内部响应再转回调用方期待的格式

对你做 `bifrost-lite` 来说，这一层就是“网关壳子”。

### 2. `core/`：统一执行引擎
这一层负责“真正跑请求”。

- 维护 `Bifrost` 主对象
- 管理 provider 列表、请求队列、worker、对象池
- 统一调用不同 provider
- 执行插件链、MCP 工具调用、错误处理

对你来说，这一层是最值钱的简历点，因为它体现了你不是在写 CRUD，而是在写一个多后端调度系统。

### 3. `core/providers/`：厂商适配层
这一层负责“把统一协议翻译成各家模型厂商自己的协议”。

- OpenAI、Anthropic、Bedrock、Gemini 等各有实现
- OpenAI-compatible 厂商大量复用 openai 逻辑
- 不兼容厂商自己做 request/response 转换

这层决定了 Bifrost 为什么能统一 15+ provider。

### 4. `plugins/`：横切能力层
这一层负责“在主链路旁边插能力”。

- governance：预算、虚拟 key、访问控制
- logging：请求日志
- semanticcache：语义缓存
- telemetry / otel / maxim：观测
- mocker：测试和开发假响应

这说明 Bifrost 不是把逻辑写死在 handler 里，而是做成了平台。

### 5. `framework/`：基础设施层
这一层负责“给网关提供长期能力”。

- `configstore/`：配置存储
- `logstore/`：日志存储
- `vectorstore/`：向量存储
- `streaming/`：流式响应累积与重组
- `tracing/`：链路追踪

你可以把它理解成“支撑产品化网关的公共底座”。

### 6. `ui/`：控制台层
这一层负责“把网关变成产品”。

- Provider 管理
- 插件管理
- 治理与日志
- 配置与观测页面

这是一个很强的产品信号：Bifrost 不只是 Go SDK，也不只是一个 API server，而是带运营面的完整系统。

## 关键执行链路

这里先讲顶层链路，不进细节实现。

### 链路 A：标准 OpenAI 兼容请求
1. 客户端请求 `POST /v1/chat/completions`
2. 路由注册于 [`transports/bifrost-http/handlers/inference.go`](/mnt/windows/source/go/bifrost/transports/bifrost-http/handlers/inference.go:593)
3. handler 把 HTTP 请求转成 `BifrostContext + BifrostRequest`
4. 请求进入 [`core/bifrost.go`](/mnt/windows/source/go/bifrost/core/bifrost.go:178) 初始化好的核心引擎
5. 核心引擎根据模型和 provider 配置做选择、排队、执行
6. provider 适配层把统一请求翻译为某个厂商 API
7. 厂商响应被再转换成 Bifrost 统一响应
8. transport 层再把统一响应序列化成 OpenAI 风格 HTTP 返回

这条链路体现的是：外部协议统一，内部执行统一，底层厂商可变。

### 链路 B：SDK 兼容请求
1. 客户端走 `/openai/...`、`/anthropic/...`、`/genai/...`
2. 兼容入口由 [`transports/bifrost-http/integrations/router.go`](/mnt/windows/source/go/bifrost/transports/bifrost-http/integrations/router.go:520) 这套 `GenericRouter` 注册
3. 兼容层先解析“厂商原生格式”
4. 再转换为内部 `BifrostRequest`
5. 后续执行链路与标准 `/v1/...` 共用同一套核心引擎

这说明 Bifrost 的关键设计不是“有很多 handler”，而是“很多入口汇聚到一个核心执行面”。

### 链路 C：服务启动与装配
1. 进程从 [`transports/bifrost-http/main.go`](/mnt/windows/source/go/bifrost/transports/bifrost-http/main.go:98) 启动
2. `main()` 创建 `BifrostHTTPServer`
3. `server.Bootstrap()` 在 [`transports/bifrost-http/server/server.go`](/mnt/windows/source/go/bifrost/transports/bifrost-http/server/server.go:1147) 里加载 config、插件、日志/异步清理器、MCP 配置
4. 通过 `bifrost.Init(...)` 创建核心引擎
5. 注册 inference routes、integration routes、provider/config/plugin/ui routes
6. `server.Start()` 启动 FastHTTP 服务

你可以把这理解为三段式启动：

- 读配置
- 组引擎
- 挂路由

## 为什么是项目亮点

### 1. 它把“代理”做成了“执行平台”
很多人写的 LLM gateway 只是：

- 收请求
- 改个 base URL
- 转发

Bifrost 不是。它内部有统一 schema、provider 抽象、插件系统、MCP、日志、治理、缓存、UI、配置存储。简历上这类系统比“封装了 OpenAI SDK”高一个等级。

### 2. transport 与 core 明确分层
`transports/` 负责协议适配，`core/` 负责执行引擎。

这意味着：

- 能同时支持标准 `/v1/...`
- 能支持厂商兼容路由
- 未来能换别的 transport，而不用重写执行逻辑

这是很标准、很强的系统设计信号。

### 3. 多 provider 统一抽象非常适合写进简历
统一多家 provider 最大的难点不是“调 API”，而是：

- 请求结构不一样
- 流式协议不一样
- 错误格式不一样
- 功能覆盖不一样

Bifrost 把这些差异压进 provider 层，对上提供统一接口。这是非常典型的“复杂性下沉”。

### 4. 它已经有平台化产品形态
从当前仓库能直接看出它不是 demo：

- HTTP gateway
- UI
- 插件
- 配置存储
- 日志与指标
- MCP server / tool execution

这使得你学习它时，不只是学一段代码，而是在学一个 AI infra 产品怎么长成型。

## 结合源码看顶层地图

### 建议先读的文件
- [`README.md`](/mnt/windows/source/go/bifrost/README.md:1)
- [`transports/bifrost-http/main.go`](/mnt/windows/source/go/bifrost/transports/bifrost-http/main.go:1)
- [`transports/bifrost-http/server/server.go`](/mnt/windows/source/go/bifrost/transports/bifrost-http/server/server.go:1147)
- [`core/bifrost.go`](/mnt/windows/source/go/bifrost/core/bifrost.go:1)
- [`transports/bifrost-http/handlers/inference.go`](/mnt/windows/source/go/bifrost/transports/bifrost-http/handlers/inference.go:593)
- [`transports/bifrost-http/integrations/router.go`](/mnt/windows/source/go/bifrost/transports/bifrost-http/integrations/router.go:520)

### 当前仓库里你要注意的一个事实
你给我的 AGENTS 说明里提到 `core/inference.go`，但我按当前仓库实际检索，没有这个文件；当前架构信息仍然主要能从 `core/bifrost.go`、`transports/bifrost-http/server/server.go`、`handlers/inference.go` 和 `integrations/router.go` 还原出来。

这件事本身也很重要：你以后做源码学习或面试表达时，要优先以当前代码为准，而不是只背文档。

## 关键 struct / 模块

### `BifrostHTTPServer`
位置：[`transports/bifrost-http/server/server.go`](/mnt/windows/source/go/bifrost/transports/bifrost-http/server/server.go:95)

它是 HTTP 产品壳。

负责：
- 持有 `Client *bifrost.Bifrost`
- 持有配置、router、websocket handler
- bootstrap 全部运行时依赖
- 注册 API/UI 路由

你可以把它理解成“应用装配器”。

### `Bifrost`
位置：[`core/bifrost.go`](/mnt/windows/source/go/bifrost/core/bifrost.go:56)

它是核心执行引擎。

负责：
- provider 管理
- request queue 管理
- plugin 管理
- channel / pool / tracing / MCP manager

你以后做 `bifrost-lite` 时，最先值得自己实现的就是这个对象。

### `CompletionHandler`
位置：[`transports/bifrost-http/handlers/inference.go`](/mnt/windows/source/go/bifrost/transports/bifrost-http/handlers/inference.go:35)

它是标准 OpenAI 风格入口。

价值不在“if else 处理请求”，而在于：
- 把 HTTP 世界转成统一内部世界
- 把路由清楚地挂到一个 handler 集合上

### `GenericRouter`
位置：[`transports/bifrost-http/integrations/router.go`](/mnt/windows/source/go/bifrost/transports/bifrost-http/integrations/router.go:520)

这是兼容层的关键亮点。

它说明 Bifrost 不是每家 SDK 都手写一套 handler，而是抽象成：
- route config
- request converter
- response converter
- error converter
- stream config

这非常适合你在简历里讲“抽象能力”。

## 为什么这个分层值得你模仿

### 适合逐步扩展
如果你做 `bifrost-lite`，完全可以按下面顺序长：

1. 一个 HTTP 入口
2. 一个核心引擎对象
3. 一个 provider 接口
4. 一个 OpenAI provider 实现
5. 一个 logging hook
6. 一个 fallback provider

这套路线和 Bifrost 真正的结构是对齐的，不会走偏成“脚本式拼接”。

### 适合面试表达
你以后可以很自然地讲成：

- transport layer
- orchestration core
- provider adapters
- plugin pipeline
- control plane

这种表述比“我做了个 AI API 转发器”强很多。

## 我自己这一课要实现什么

你的作业不要上来就写全功能网关。第一课只做“产品骨架”。

### 这节课你的 bifrost-lite 目标
实现一个最小的目录与模块划分：

```text
bifrost-lite/
├── cmd/server/main.go
├── internal/transport/http/
├── internal/core/
├── internal/providers/
├── internal/plugins/
└── README.md
```

### 第一课你要亲手完成的内容
1. 画一张顶层架构图
2. 写一段 1 分钟项目介绍
3. 建立上面的目录骨架
4. 在 `README.md` 写清楚每层职责
5. 先不要接真实 provider，只把系统边界定义出来

### 这一课的完成标准
当别人问你“bifrost-lite 是什么”，你能直接回答：

“它是一个受 Bifrost 启发的多提供商 AI Gateway。HTTP transport 接住 OpenAI 风格请求，core 负责统一调度，provider adapters 负责翻译不同模型厂商协议，plugins 负责治理/日志/缓存等横切能力。”

如果你能稳定说出这段，你已经进入正确学习轨道了。

## 为什么这节课对简历有帮助

你现在还不能说“我做了一个和 Bifrost 一样大的系统”，但你可以开始积累非常诚实的表达：

- studied and reproduced the top-level architecture of a multi-provider AI gateway inspired by Bifrost
- designed a bifrost-lite skeleton with transport/core/provider/plugin separation
- mapped OpenAI-compatible entrypoints to a unified internal orchestration layer

把这三件事做实，后面每一课都能继续往简历上加可信细节。

## 下一课
Lesson 2 我们应该进入“启动与装配”：

- `main.go` 怎么启动
- `Bootstrap()` 到底装了哪些运行时对象
- config / plugin / core client 是怎么串起来的

那一课结束后，你就该开始写自己的 `cmd/server/main.go` 和 `internal/app/bootstrap.go` 了。
