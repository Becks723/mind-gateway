# Lesson 2: 启动与装配

## 这一课学完你能做什么
你可以把 Bifrost 的启动过程讲成一条清晰的工程链路：进程如何启动、配置如何加载、依赖如何组装、核心引擎何时创建、路由何时挂载，以及为什么这套 bootstrap 设计对 `bifrost-lite` 很关键。

## 这一层的职责
Bifrost 的 bootstrap 层不是“顺手在 `main()` 里 new 一堆对象”，而是专门负责把一个 AI gateway 运行起来。

这一层主要做 5 件事：

1. 读取启动参数，决定进程如何启动
2. 加载配置与持久化状态，恢复运行时世界
3. 实例化核心引擎 `Bifrost`
4. 组装 HTTP server、middleware、routes、UI、WebSocket、tracing
5. 管理进程生命周期，包括优雅关闭

如果上一课讲的是“系统长什么样”，这一课讲的就是“它怎么活起来”。

## 启动链路先看全景

### 顶层调用顺序
1. [`transports/bifrost-http/main.go`](/mnt/windows/source/go/bifrost/transports/bifrost-http/main.go:117) 解析 flag，设置 logger
2. `main()` 调用 `server.Bootstrap(ctx)`
3. [`transports/bifrost-http/server/server.go`](/mnt/windows/source/go/bifrost/transports/bifrost-http/server/server.go:1147) 的 `Bootstrap()` 加载配置、插件、stores、MCP、client、routes
4. `main()` 再调用 `server.Start()`
5. `Start()` 绑定监听端口，启动 FastHTTP，并处理信号和 graceful shutdown

这就是 Bifrost 的真实装配流程。

## 关键执行链路

### 链路 A：`main.go` 只做最上层控制
入口在 [`transports/bifrost-http/main.go`](/mnt/windows/source/go/bifrost/transports/bifrost-http/main.go:88)。

它做的事情相对克制：

1. 初始化 `BifrostHTTPServer`
2. 从命令行 flag 把 `host`、`port`、`app-dir`、`log-level` 注入 server
3. 初始化 logger 并分发给 `lib`、`server`、`handlers`
4. 执行 `Bootstrap`
5. 执行 `Start`

这个设计有一个明显优点：`main.go` 不是业务中心，真正的装配逻辑都被推到 `BifrostHTTPServer`。

这对可维护性很重要，因为：

- CLI 启动逻辑可以很薄
- 测试和复用更容易
- server 作为“应用对象”更清晰

### 链路 B：`LoadConfig()` 先把“状态世界”搭起来
最关键的配置入口是 [`transports/bifrost-http/lib/config.go`](/mnt/windows/source/go/bifrost/transports/bifrost-http/lib/config.go:375) 的 `LoadConfig()`。

它不是只读 `config.json`，而是按顺序创建整个运行时配置状态：

1. 计算 `config.json`、`config.db`、`logs.db` 路径
2. 读取并校验 `config.json`
3. 初始化 encryption
4. 初始化 stores：`ConfigStore`、`LogsStore`、`VectorStore`
5. 初始化 KV store
6. 加载 client config
7. 加载 providers
8. 加载 MCP config
9. 加载 governance config
10. 加载 auth config
11. 加载 plugins
12. 初始化 framework config 和 model catalog
13. 设置 websocket 默认配置

这一步特别值得你注意：Bifrost 把“配置”设计成了一个恢复运行时状态的过程，而不是一次性 parse JSON。

### 链路 C：配置来源不是单一文件，而是“文件 + 持久化 store + 默认值”
`LoadConfig()` 里有一个很重要的工程思想：配置是多源合并的。

比如 `client config`：

- 先尝试从 `ConfigStore` 读取
- 如果没有，再看 `config.json`
- 再不行，落到默认值

provider 配置也是类似思路：

- 优先使用持久化 store 中的状态
- 再用 `config.json` 做 hash-based reconciliation
- 如果两边都没有，甚至可以从环境变量自动探测 provider

这意味着 Bifrost 的配置系统不是死板静态配置，而是支持：

- UI 改配置
- 配置持久化
- 文件覆盖
- 启动时状态恢复

这就是它像“产品”而不是“脚本”的地方。

### 链路 D：`Bootstrap()` 把“配置状态”接成“运行中的系统”
真正的装配核心在 [`server.go`](/mnt/windows/source/go/bifrost/transports/bifrost-http/server/server.go:1147) 的 `Bootstrap()`。

它大致按这个顺序工作：

1. 创建 `BifrostContext`
2. 确保 app directory 存在
3. 调用 `lib.LoadConfig(...)`
4. 提前初始化 `WebSocketHandler`
5. 初始化 plugin loader
6. 初始化日志清理器和异步 job 清理器
7. 调用 `LoadPlugins(ctx)`
8. 如果条件满足，初始化 async job executor
9. 组装 `MCPConfig`
10. 用 `bifrost.Init(...)` 创建核心引擎 `s.Client`
11. 同步 plugin 顺序到 core
12. 遍历 provider，拉模型并同步到 model catalog
13. `s.Config.SetBifrostClient(s.Client)`
14. 创建 router
15. 准备 common/api/inference middlewares
16. 注册 API routes
17. 注册 inference routes
18. 注册 UI routes
19. 创建最终的 `fasthttp.Server`

你会发现这已经不是“HTTP server startup”，而是完整应用 runtime assembly。

## 为什么先 `LoadConfig` 再 `bifrost.Init`
这是这节课最关键的设计理解。

`bifrost.Init(...)` 需要的不是原始 JSON，而是已经被整理过的运行时依赖：

- `Account`
- `LLMPlugins`
- `MCPPlugins`
- `MCPConfig`
- `KVStore`
- `Logger`

这些东西都来自 `LoadConfig()` 之后的 `s.Config`。

换句话说：

- `LoadConfig()` 负责把“配置”变成“可注入的依赖”
- `bifrost.Init()` 负责把这些依赖变成“执行引擎”

这是非常标准的依赖装配思路，值得你在 `bifrost-lite` 里照搬。

## 路由装配不是一坨，而是分成三类

### 1. API routes
由 [`RegisterAPIRoutes`](/mnt/windows/source/go/bifrost/transports/bifrost-http/server/server.go:976) 注册。

这类路由更像控制面：

- health
- providers
- mcp
- config
- oauth
- plugins
- session
- prompts
- cache
- governance
- logging
- websocket
- metrics

这说明 Bifrost 不只是 serving inference，也有管理面和运营面。

### 2. inference routes
由 [`RegisterInferenceRoutes`](/mnt/windows/source/go/bifrost/transports/bifrost-http/server/server.go:953) 注册。

这里会创建：

- `CompletionHandler`
- `IntegrationHandler`
- `MCPInferenceHandler`
- `MCPServerHandler`
- `AsyncHandler`

也就是说，真正的推理入口不止一个：

- 标准 `/v1/...`
- 各家 SDK 兼容入口
- MCP 工具与 MCP server
- async inference

### 3. UI routes
最后才由 [`RegisterUIRoutes`](/mnt/windows/source/go/bifrost/transports/bifrost-http/server/server.go:1078) 注册。

顺序上特意放后面，是因为 UI 常常要兜底接未命中的前端路由。

这类顺序意识非常像成熟 Web 产品，不像简单 API demo。

## 中间件装配也体现了产品思路

### common middleware
`PrepareCommonMiddlewares()` 在 [`server.go`](/mnt/windows/source/go/bifrost/transports/bifrost-http/server/server.go:1123) 里先尝试挂 telemetry middleware。

说明观测不是后来补上的，而是默认进入主链路。

### API 与 inference middleware 分离
`Bootstrap()` 里把中间件拆成：

- `apiMiddlewares`
- `inferenceMiddlewares`

然后按用途分别挂载。

这很合理，因为：

- 管理路由与推理路由的鉴权策略不同
- 推理路由往往还要多一层 transport interceptor 和 tracing

### tracing 是在 inference 路由前装上的
在注册 inference 路由之前，Bifrost 会：

1. 收集 observability plugins
2. 初始化 `TraceStore`
3. 创建 tracer
4. `s.Client.SetTracer(tracer)`
5. 创建 `TracingMiddleware`
6. 把 tracing middleware 加进 inference 链路

这意味着 trace 不是 provider 内部孤立打点，而是从 transport 层开始贯穿。

这就是“网关级 observability”的设计。

## 为什么这是项目亮点

### 1. 启动过程本身就展示了平台化能力
一个普通代理的启动逻辑通常只有：

- 读 env
- 起 HTTP server

而 Bifrost 的启动链路包含：

- config store
- logs store
- vector store
- plugin loader
- websocket
- tracing
- MCP
- async job executor

这说明它是一个有状态、有扩展点、有控制面的平台。

### 2. `BifrostHTTPServer` 是一个很强的装配中心
`BifrostHTTPServer` 不是单纯的 server wrapper，而是应用级 runtime container。

它持有：

- `Client *bifrost.Bifrost`
- `Config *lib.Config`
- `Router *router.Router`
- `Server *fasthttp.Server`
- websocket / mcp / tracing / auth 等 handler 和 middleware

你在简历里如果能自己实现一个类似的装配中心，会比“我有几个 handler”强很多。

### 3. 配置合并策略很成熟
Bifrost 没有简单粗暴地“文件覆盖一切”，而是做了：

- DB / file / default 合并
- hash-based reconciliation
- UI 修改保留
- 启动时恢复

这是一种非常真实的后台产品能力。

### 4. graceful shutdown 做得完整
`Start()` 不只是 `ListenAndServe`。

它会在接收到信号后：

1. 关闭 HTTP server
2. cancel 主 context
3. `s.Client.Shutdown()`
4. 停止 logs cleaner
5. 停止 async job cleaner
6. 停止 ws ticket store
7. 关闭 websocket pool
8. `s.Config.Close(shutdownCtx)`

而 [`Config.Close(...)`](/mnt/windows/source/go/bifrost/transports/bifrost-http/lib/config.go:2403) 又会继续回收：

- model catalog
- token refresh worker
- KVStore
- ConfigStore
- LogsStore
- VectorStore

这说明作者很在意后台 goroutine 和资源生命周期，不是“Ctrl+C 就算了”。

## 关键文件与符号

### `main()`
位置：[`transports/bifrost-http/main.go`](/mnt/windows/source/go/bifrost/transports/bifrost-http/main.go:117)

作用：
- 启动入口
- logger 与 flag 控制
- 调 `Bootstrap` 与 `Start`

### `NewBifrostHTTPServer`
位置：[`transports/bifrost-http/server/server.go`](/mnt/windows/source/go/bifrost/transports/bifrost-http/server/server.go:141)

作用：
- 创建 server 容器对象
- 承接 Version、UI、host/port/app-dir 等初始参数

### `LoadConfig`
位置：[`transports/bifrost-http/lib/config.go`](/mnt/windows/source/go/bifrost/transports/bifrost-http/lib/config.go:375)

作用：
- 从文件和 store 重建运行时配置
- 初始化 stores / kv / providers / plugins / framework

### `Bootstrap`
位置：[`transports/bifrost-http/server/server.go`](/mnt/windows/source/go/bifrost/transports/bifrost-http/server/server.go:1147)

作用：
- 用配置组装整个运行时系统
- 创建 core client
- 注册全部 routes

### `Start`
位置：[`transports/bifrost-http/server/server.go`](/mnt/windows/source/go/bifrost/transports/bifrost-http/server/server.go:1373)

作用：
- 真正对外提供服务
- 做信号监听和优雅关闭

## 对 `bifrost-lite` 的重建建议

### 你不要直接复制 Bifrost 的复杂度
你的 lite 版现在不需要：

- config store
- logs store
- vector store
- websocket
- MCP
- plugin loader

但你应该复制它的装配思想。

### 你的最小 bootstrap 结构
建议你做成下面这样：

```text
cmd/server/main.go
internal/app/bootstrap.go
internal/config/config.go
internal/transport/http/server.go
internal/core/gateway.go
```

### 你这一课要实现的最小职责分配

#### `cmd/server/main.go`
- 解析 `PORT`
- 初始化 logger
- 调 `app.Bootstrap()`
- 调 `server.Start()`

#### `internal/config/config.go`
- 读本地配置文件或 env
- 返回一个 `Config` struct

#### `internal/app/bootstrap.go`
- 调用 `LoadConfig()`
- 初始化 provider list
- 初始化 gateway core
- 初始化 HTTP server
- 注册 routes
- 返回 app/server

#### `internal/transport/http/server.go`
- 持有 router 与 gateway core
- 暴露 `Start()` 和 `Shutdown()`

这就是你的 Lesson 2 作业版本。

## 我自己这一课要实现什么

### 目标
做出一个“能启动、能关停、能承载核心对象”的应用骨架。

### 你这节课要亲手写的东西
1. `main.go`
2. `bootstrap.go`
3. `config.go`
4. `server.go`
5. 一个最小 `/health` 路由

### 验收标准
你自己的 `bifrost-lite` 应该做到：

1. 启动时打印当前配置
2. 创建一个 `Gateway` 核心对象
3. 挂一个 HTTP server
4. 至少暴露 `/health`
5. 接收到 `SIGINT` 时能优雅退出

如果你能完成这 5 件事，你已经不再是在写“单文件 demo”，而是在写一个真正可扩展的网关壳。

## 简历怎么说才诚实

- built a Bifrost-inspired gateway bootstrap layer with explicit config loading, dependency assembly, and graceful shutdown
- implemented a lightweight runtime container that wires transport, core gateway, and provider configuration
- reproduced the startup lifecycle of a multi-provider AI gateway, including route registration and process-level shutdown handling

这些表述是可信的，因为它们说的是“受启发地复现架构”，不是冒充你写了 Bifrost 本体。

## 下一课
Lesson 3 应该进入 `core/bifrost.go`，专门拆核心执行引擎：

- `Bifrost` 主对象到底管什么
- queue / worker / pool 是怎么服务 provider 的
- 为什么这是 bifrost-lite 最值得亲手实现的一层
