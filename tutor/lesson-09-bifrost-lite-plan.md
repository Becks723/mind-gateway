# Lesson 9: 把学习收束成一个 bifrost-lite

## 这一课学完你能做什么
你会得到一份可以直接开工的 `bifrost-lite` 实施路线图：知道先做什么，哪些先不做，做到什么程度已经足够写进简历。

## 这一层的职责
这一课的职责不是讲一个单独模块，而是把前 8 节学到的东西压缩成一个“小而真”的工程方案。

你现在最需要的不是再看更多源码，而是回答五个现实问题：

1. `bifrost-lite` 的最小可行架构是什么
2. 第一版必须做哪些能力
3. 哪些能力先明确不做
4. 每一阶段完成后可以怎么写进简历
5. 怎么避免把项目做成一个“功能很多但没有主线”的半成品

## 我先给你结论
如果你的目标是“丰富简历 + 做出一个 Bifrost 风格的作品”，那 `bifrost-lite` 第一版应该只保留这五层：

1. `HTTP transport`
2. `core request engine`
3. `provider abstraction`
4. `retries + fallback`
5. `plugin hooks`

在这之上，再选两项高级能力作为第二阶段：

1. `tool loop (MCP-inspired)`
2. `governance lite`

这就是最划算的范围。

如果你按这个范围做完，它已经不是 demo，而是一个可信的 AI gateway 项目。

## 最小可信架构

### 这一层的职责
`bifrost-lite` 不需要照搬原仓库的大体量，但必须保留 Bifrost 最核心的 architectural shape。

也就是说，你的 clone 最少也要让人一眼看出：

- transport 和 core 分离
- core 不直接依赖某个 provider
- provider behind interface
- 重试 / fallback 在 orchestration 层，而不是写死在 handler
- 插件能力用于承载 cross-cutting concerns

### 建议目录
我建议你直接做一个比 Bifrost 小很多，但结构清楚的版本：

```text
bifrost-lite/
├── cmd/server/main.go
├── internal/core/
│   ├── gateway.go
│   ├── queue.go
│   ├── retry.go
│   └── fallback.go
├── internal/providers/
│   ├── provider.go
│   ├── openai/
│   └── mock/
├── internal/http/
│   ├── server.go
│   ├── handlers.go
│   └── context.go
├── internal/plugins/
│   ├── plugin.go
│   ├── logging.go
│   └── governance.go
├── internal/tools/
│   ├── registry.go
│   └── loop.go
├── internal/config/
│   └── config.go
└── README.md
```

### 为什么是项目亮点
这个结构的好处是你能明确展示：

- 你不是只会写一个 handler 然后转发请求
- 你知道如何把 orchestration、provider adapter、cross-cutting concerns 拆开
- 你的项目天然能继续演化，而不是只能支撑单文件 demo

### 我自己这一课要实现什么
这一课你要先把仓库蓝图定下来，哪怕先只建目录和接口也可以。

第一步不是实现功能，而是把边界先写死：

1. `Gateway` 负责 orchestration
2. `Provider` 负责外部模型调用
3. `Handler` 只负责 HTTP 入参与出参
4. `Plugin` 负责前后置拦截
5. `ToolRegistry` 负责本地工具调用

## Phase 1：MVP 必做能力

### 这一层的职责
第一阶段的目标只有一个：做出一个真的能用的 gateway。

这里的“能用”定义很简单：

- 能收一个 OpenAI-compatible chat 请求
- 能转发到一个 provider
- 能返回结果
- provider 出错时可以 retry / fallback
- 能留出后续扩展点

### 关键执行链路
MVP 的主链路应该是：

1. `POST /v1/chat/completions`
2. handler 解析请求并生成内部 request
3. `Gateway.HandleChat(...)`
4. pre-hooks 执行
5. 选择 provider / key
6. 调用 `Provider.ChatCompletion(...)`
7. 如果失败，进入 retry
8. retry 仍失败则尝试 fallback
9. post-hooks 执行
10. 返回 response

### 为什么是项目亮点
做到这里，你已经拥有 Bifrost 最核心的“网关本体”了。

这和“只是代理 OpenAI”完全不同，因为你的系统已经有：

- internal request abstraction
- orchestration layer
- resilience logic
- extensibility points

### 我自己这一课要实现什么
MVP 我建议你只做下面这些，不多也不少：

1. 一个 `OpenAIProvider`
2. 一个 `MockProvider`
3. 一个 chat completions endpoint
4. 同步非流式响应
5. 最基础的 retry
6. 一个 fallback provider 配置
7. 一个 logging plugin

如果你时间够，再加 streaming；如果时间不够，先别碰 streaming。

## Phase 2：把它从 proxy 提升成 gateway

### 这一层的职责
第二阶段的目标是补上“为什么这个项目值得写进简历”的部分。

我建议只补两件事：

1. `tool loop`
2. `governance lite`

### 关键执行链路

#### A. Tool loop
1. 请求携带 `tools`
2. gateway 在调用 provider 前把 tools 注入模型请求
3. 模型返回 tool call
4. gateway 命中本地 `ToolRegistry`
5. 执行 tool
6. 把 tool result 回送模型
7. 得到最终答案

#### B. Governance lite
1. 请求 header 携带 `virtual key`
2. pre-hook 校验 key 是否存在、是否超限
3. 请求通过后进入 provider 调用
4. post-hook 记录 usage
5. 下次请求按额度与规则继续决策

### 为什么是项目亮点
这两项能力组合起来，能让你的项目一下子跨过两个门槛：

1. 从 `model proxy` 变成 `agent-aware gateway`
2. 从 `个人 demo` 变成 `multi-tenant infra prototype`

这正是简历上最值钱的部分。

### 我自己这一课要实现什么
第二阶段你不要做“很多高级功能”，只做下面这些就够：

1. 一个本地工具注册表
2. 一个简单的 tool execution loop
3. 一个 `virtual key -> budget` 映射表
4. request/token usage 记录
5. 超限拒绝

做到这里，`bifrost-lite` 就已经非常像一个真正的基础设施项目了。

## Phase 3：观测与展示

### 这一层的职责
第三阶段不是先追求更多功能，而是让这个项目“更像你自己能解释清楚、演示清楚、面试里讲清楚的作品”。

### 关键执行链路
这一层最重要的是让每次请求都带上这些信息：

1. `request_id`
2. `selected_provider`
3. `retry_count`
4. `fallback_index`
5. `latency_ms`
6. `virtual_key`

你可以先把这些写到结构化日志里，再决定要不要上简单 dashboard。

### 为什么是项目亮点
面试官最容易判断你是不是“真的做过 infra”的方式，不是看你会不会调 API，而是看你能不能回答：

- 某个请求为什么慢
- 为什么发生 fallback
- 哪个 key 被选中了
- 某个租户花了多少 token

可观测性让这些问题有证据链。

### 我自己这一课要实现什么
第三阶段你只需要做两件事：

1. 结构化日志
2. 一个简单的 `/debug/requests` 或统计 endpoint

不要急着上复杂 tracing 系统。先让你的 gateway 可解释，再谈完整 observability。

## 明确先不做什么

### 这一层的职责
项目能做完，靠的不只是会加功能，更靠会砍功能。

### 为什么是项目亮点
范围控制本身就是工程能力。你要做的是一个“能完成、能展示、能复盘”的作品，而不是把 Bifrost 缩写一遍。

### 我自己这一课要实现什么
`bifrost-lite` 第一版我建议明确先不做：

1. 完整 UI control plane
2. 多数据库 config store
3. 20+ providers
4. 复杂 semantic cache
5. 完整 MCP server protocol
6. 企业级 RBAC / team / customer 管理
7. 大规模热更新和原子配置交换
8. 全量 Responses API / batch / file APIs

你现在真正需要的是一个收敛版本：

- 一个主接口
- 两个 provider
- 一个插件系统
- 一套 retry/fallback
- 一个 tool loop
- 一个 governance lite

这已经足够强。

## 关键里程碑

### 这一层的职责
里程碑的作用是把“想做一个项目”变成“每周都能完成一个可见成果”。

### 我建议的节奏

1. `Milestone 1`
   完成 HTTP server、chat endpoint、OpenAI provider、Mock provider

2. `Milestone 2`
   完成 internal request schema、Gateway orchestration、retry、fallback

3. `Milestone 3`
   完成 plugin hooks、logging plugin、request metadata 注入

4. `Milestone 4`
   完成 tool registry、tool loop、至少两个本地工具

5. `Milestone 5`
   完成 governance lite、virtual key、usage accounting、budget reject

6. `Milestone 6`
   完成结构化日志、演示文档、架构图、README case study

### 为什么是项目亮点
这个顺序是故意设计的：

- 先保证主链路能跑
- 再保证它有 gateway 味道
- 再补足最有简历价值的高级能力
- 最后做展示与包装

这比一开始做 UI、数据库、很多 providers 更稳。

### 我自己这一课要实现什么
你现在可以把这 6 个里程碑直接抄成 issue list 或 project board。

每完成一个 milestone，就更新一次 README 和简历表达。

## 简历上应该怎么写

### 为什么是项目亮点
同一个项目，写法不同，含金量差很多。

你不该把它写成：

- built an AI proxy for OpenAI-compatible APIs

这种写法太弱。

你应该写成更接近系统设计成果的表达。

### 推荐表达

- built `bifrost-lite`, a Bifrost-inspired multi-provider AI gateway with provider abstraction, retry/fallback orchestration, and plugin-based request interception
- added an MCP-inspired tool execution loop that turned static chat requests into tool-calling agent workflows
- implemented governance-lite capabilities including virtual keys, usage accounting, and budget-based request enforcement
- added structured observability for provider selection, retries, fallbacks, and per-request latency

## 最后给你的项目判断标准

### 这一层的职责
这一节最后要解决的是“做到什么程度就可以停”。

### 我给你的判断标准
如果你的 `bifrost-lite` 已经满足下面这些条件，就已经足够写进简历和作品集：

1. 有清晰的 transport / core / provider / plugin 分层
2. 至少支持两个 provider，其中一个可以是 mock
3. 有 retry 和 fallback
4. 有 plugin hooks
5. 有一个 tool loop
6. 有一个 governance lite
7. 有结构化日志和 README 架构图

满足这七条，它就不是“简化版玩具”，而是“有明显架构主张的基础设施作品”。

## 下一课
如果继续往下学，最自然的 Lesson 10 就不再是源码讲解，而是直接开始落地：

`bifrost-lite` 第 1 周实现计划。

也就是把上面的 milestone 再拆成：

1. 第一天建什么文件
2. 第一周先写哪些接口
3. 哪些测试最先补
4. 哪个 demo 最快能跑起来
