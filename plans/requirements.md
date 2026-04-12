# mind-gateway 需求文档

## 1. 项目目标

`mind-gateway` 是一个受 Bifrost 启发、但范围明确收敛的多 Provider AI Gateway。

项目目标不是照搬 `bifrost/`，而是做出一个可以独立讲清楚、可以本地运行、可以稳定演示、可以写进简历的实现版作品。

目标产物应满足以下四点：

- 对外提供 OpenAI-compatible 的 `/v1/chat/completions` 接口。
- 对内具备独立的 gateway orchestration，而不是简单 HTTP 反向代理。
- 至少体现两类“网关味道”能力：可靠性能力与扩展能力。
- 具备完整文档、测试、演示脚本与简历表述素材。

## 2. 项目定位

`mind-gateway` 的定位是 `Bifrost-inspired gateway prototype`，重点展示以下能力：

- transport / core / provider 明确分层
- 多 provider 统一抽象
- 请求队列、worker、重试、fallback
- plugin hook 扩展点
- tool loop 与 governance lite
- 结构化日志与可观测调试接口

不追求第一版覆盖 Bifrost 的完整产品面，如 UI、20+ providers、复杂 semantic cache、完整 MCP server 协议、企业级控制面。

## 3. 成功标准

项目完成时，应达到以下标准：

1. 可以本地启动一个 Go 服务并稳定监听 HTTP 端口。
2. 可以通过 `/v1/chat/completions` 跑通非流式请求。
3. 至少支持 `mock` 与 `openai` 两类 provider。
4. provider 失败时具备 retry 与 fallback 能力。
5. 具备最小插件体系，至少包含 `logging` 与 `governance` 两个插件。
6. 支持最小流式响应能力。
7. 支持最小 tool loop，能执行本地注册工具。
8. 有集成测试、演示脚本、样例配置和 README。
9. 可以产出 2 到 3 条真实、不过度夸大的简历表述。

## 4. 用户画像与使用场景

主要使用者：

- 需要统一接入不同 LLM Provider 的后端开发者
- 需要一个本地可控 AI gateway 的个人开发者
- 面试中需要展示 AI infra / gateway 能力的作者本人

核心使用场景：

1. 本地启动 gateway，通过统一接口切换 provider。
2. 当主 provider 失败时自动重试，并在必要时切换到 fallback provider。
3. 通过 virtual key 做最小预算治理。
4. 通过本地工具执行 loop 展示 agent-aware gateway 能力。
5. 通过结构化日志与 debug 接口解释一次请求的执行轨迹。

## 5. 功能范围

### 5.1 MVP 必做

- HTTP 服务启动与优雅关闭
- `/healthz`
- `/v1/chat/completions`
- 统一的内部 request / response schema
- `Provider` 抽象接口
- `mock` provider
- `openai` provider
- provider registry
- queue + worker 执行模型
- retry + backoff
- fallback provider 链路
- logging plugin
- 基础配置加载
- 基础集成测试

### 5.2 第二阶段必做

- SSE streaming chat completions
- governance lite
- tool registry
- tool loop
- `/debug/requests` 或 `/debug/stats`
- 请求级结构化日志

### 5.3 明确不做

- UI 控制台
- 数据库存储层
- 完整 MCP server protocol
- 复杂 semantic cache
- 多租户 RBAC
- 复杂插件 marketplace
- 多模态全接口覆盖
- 大规模 provider 适配

## 6. 架构约束

项目结构建议如下：

```text
mind-gateway/
├── cmd/server/main.go
├── internal/config/
├── internal/transport/http/
├── internal/core/
├── internal/provider/
├── internal/plugin/
├── internal/tools/
├── internal/observability/
├── testdata/
├── scripts/
├── plans/
└── README.md
```

架构原则：

1. `transport` 只做协议解析、校验、序列化，不承载调度策略。
2. `core` 负责 orchestration、队列、重试、fallback、plugin pipeline。
3. `provider` 负责外部模型请求的适配与标准化。
4. `plugin` 负责横切能力，不把治理和日志写死在主链路中。
5. `tools` 负责本地工具注册和执行，不与 provider 耦合。

## 6.1 冻结实现决策

下面这些内容视为当前版本的固定实现方向，后续实现默认不再反复改名或改层级：

1. 主语言固定为 `Go`。
2. 服务形态固定为单进程 HTTP gateway。
3. 第一版只暴露 `GET /healthz`、`POST /v1/chat/completions`、`GET /debug/requests`。
4. 第一版主接口只做 `chat completions`，不并行铺开 embeddings、images、responses。
5. provider 固定先做 `mock` 与 `openai`，其他 provider 一律延后。
6. 调度模型固定为 `provider queue + worker goroutine`，不做全局 worker pool。
7. 插件模型固定为 `PreHook + PostHook`，不再额外设计多套 middleware 体系。
8. tool 能力固定为“本地 tool registry + 单轮/多轮 tool loop”，不实现完整 MCP server。
9. governance 固定为 `virtual key + request/token budget`，不展开 team、project、RBAC。
10. debug 能力固定为“内存中保留最近请求摘要”，不接数据库。

## 7. 核心功能需求

### 7.1 HTTP Transport

- 提供 `POST /v1/chat/completions`
- 提供 `GET /healthz`
- 提供 `GET /debug/requests` 或同类调试接口
- 支持 JSON 非流式返回
- 第二阶段支持 SSE streaming

验收要求：

- 能用 `curl` 直接调用
- 错误返回结构统一
- 请求超时、取消、非法参数有清晰响应

### 7.2 Core Gateway

- 提供统一 `Gateway` 对象
- 支持 provider 注册、查找与选择
- 支持请求入队与 worker 消费
- 支持执行前后插件 hook
- 支持 retry / backoff / fallback
- 支持记录 request id、provider、retry count、latency

验收要求：

- 同一个 handler 不直接调用 provider 实现
- 核心链路可在日志中追踪
- 关闭服务时 worker 能优雅退出

### 7.3 Provider Abstraction

- 统一 `Provider` 接口
- `mock` provider 用于本地稳定测试
- `openai` provider 用于真实外部联调
- provider 响应转换为统一 schema

验收要求：

- 至少两个 provider 通过同一接口工作
- mock provider 能稳定返回固定结构
- openai provider 能通过配置接入

### 7.4 Reliability

- 支持最大重试次数配置
- 支持退避时间配置
- 支持 fallback provider 列表
- 支持队列长度和 worker 数量配置
- 队列过载时返回明确错误

验收要求：

- 人为制造 provider 错误时，日志能看到 retry 次数
- 主 provider 故障时，请求能切换到 fallback 成功返回
- 队列压满时系统不 panic

### 7.5 Plugin System

- 设计 `PreHook` / `PostHook`
- logging plugin 记录请求输入、输出、耗时、provider、错误
- governance plugin 在请求前做额度校验，在请求后记账

验收要求：

- 插件注册顺序与执行顺序清晰
- logging 插件可单独启停
- governance 插件能拒绝超限请求

### 7.6 Tool Loop

- 支持本地工具注册
- 支持模型返回 tool call 后执行本地工具
- 支持将 tool result 追加回对话再请求模型

验收要求：

- 至少 2 个本地工具
- 至少 1 条集成测试覆盖工具回路
- 能做一段演示：用户提问 -> 调工具 -> 得到最终答案

## 8. 非功能需求

### 8.1 可测试性

- 单元测试覆盖核心 schema、provider adapter、plugin、governance
- 集成测试覆盖主链路
- 核心 happy path 与 fallback path 必须有自动化验证

### 8.2 可观测性

- 每个请求生成 `request_id`
- 输出结构化日志
- 至少记录 `provider`、`retry_count`、`fallback_index`、`latency_ms`
- 暴露最小调试接口查看近期请求

### 8.3 可演示性

- 提供 `README` 启动说明
- 提供样例配置
- 提供 `curl` 示例
- 提供 `scripts/demo.sh` 或等价演示脚本

### 8.4 可维护性

- 包结构清晰
- 接口边界明确
- 避免业务代码全塞进 `main.go`
- 配置项最少但可解释

## 9. 交付物清单

项目完成时，仓库中应至少包含：

- 可运行源码
- 样例配置
- 单元测试与集成测试
- README
- 架构说明图或文字版执行路径
- 演示脚本
- 性能/压测最小记录
- 简历表述草稿

## 9.1 代码交付清单

实现完成时，预期至少存在以下文件：

```text
cmd/server/main.go
internal/app/app.go
internal/config/config.go
internal/config/load.go
internal/core/gateway.go
internal/core/executor.go
internal/core/queue.go
internal/core/retry.go
internal/core/fallback.go
internal/core/types.go
internal/provider/provider.go
internal/provider/registry.go
internal/provider/mock/mock.go
internal/provider/openai/client.go
internal/provider/openai/adapter.go
internal/provider/openai/provider.go
internal/plugin/plugin.go
internal/plugin/pipeline.go
internal/plugin/logging/logging.go
internal/plugin/governance/governance.go
internal/tools/registry.go
internal/tools/builtin.go
internal/tools/loop.go
internal/transport/http/server.go
internal/transport/http/router.go
internal/transport/http/handlers_health.go
internal/transport/http/handlers_chat.go
internal/transport/http/handlers_debug.go
internal/transport/http/middleware.go
internal/transport/http/openai_api.go
internal/observability/debug_store.go
internal/observability/request_log.go
scripts/demo.sh
testdata/config.dev.yaml
README.md
```

## 10. 简历价值目标

项目完成后，应能诚实支持如下表述方向：

- 实现了一个受 Bifrost 启发的多 Provider AI Gateway，支持统一 OpenAI-compatible 接口。
- 设计并实现了基于 queue + worker 的请求调度内核，具备 retry、backoff 与 fallback。
- 构建了插件化扩展机制，用于日志记录、预算治理与请求审计。
- 实现了最小 tool loop，使 gateway 具备 agent-aware 的工具调用能力。

## 11. 风险与范围控制

主要风险：

- 从零实现时过早追求“大而全”
- 过早引入 UI、数据库、过多 provider，导致主线失焦
- 流式协议、tool loop、governance 同时推进，增加复杂度
- 没有可见成果，导致中途失去反馈

控制原则：

1. 先跑通非流式主链路，再做 streaming。
2. 先做 mock provider 与测试，再接真实 provider。
3. 先做 logging plugin，再做 governance plugin。
4. 每天必须产出一个可展示结果。
5. 完成“能解释、能演示、能复盘”的版本优先于继续加功能。

## 11.1 延后变更规则

为了防止实现阶段漂移，以下情况才允许调整设计：

1. 当前设计无法通过自动化测试验证。
2. 当前设计会显著增加未来实现复杂度。
3. 当前设计无法支持既定演示目标。

非上述情况，不因为“也许以后更优雅”而随意改目录、改接口、改命名。

## 12. 最终验收清单

项目视为完成，至少满足以下条件：

1. `go test ./...` 主体通过。
2. 本地可启动服务并调用 `mock` 与 `openai` 两类 provider。
3. 非流式与流式请求都可演示。
4. retry、fallback、governance、tool loop 均有一条可复现演示。
5. README 包含架构、启动、配置、演示、测试说明。
6. 可给出 3 分钟架构讲解与 5 分钟 demo。
