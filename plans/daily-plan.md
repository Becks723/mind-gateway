# mind-gateway 按天落地计划

## 节奏说明

计划按 14 天设计，目标是每天都有“可运行、可截图、可讲述”的正反馈。

执行原则：

- 每天只聚焦一个主主题，避免同时开太多口子。
- 每天结束时必须有可量化验收。
- 每天结束时必须有一个可以在面试里展示的成果点。

## Day 1：定仓库骨架与最小服务

目标：让项目从“空仓库”变成“可启动服务”。

当前审阅后补充约束：Day 1 的最小服务框架固定使用 `fasthttp`。

工作项：

- 建立 `cmd/server` 与 `internal/...` 基础目录
- 写最小 `main.go`
- 提供 `GET /healthz`
- 写 README 初版
- 写样例配置文件骨架
- 将底层 HTTP 服务从 `net/http` 固定为 `fasthttp`

当天我会实际创建的文件：

- `cmd/server/main.go`
- `internal/app/app.go`
- `internal/transport/http/server.go`
- `internal/transport/http/router.go`
- `internal/transport/http/handler/health.go`
- `internal/transport/http/handler/health_test.go`
- `README.md`
- `testdata/config.dev.yaml`

量化验收：

- 新增不少于 8 个目录
- `go run ./cmd/server` 可启动
- `curl /healthz` 返回 `200`
- 如果运行环境不允许监听端口，则至少用内存监听测试验证 `/healthz`
- 传输层基于 `fasthttp` 而不是 `net/http`
- README 至少包含启动方式和目录说明

当天正反馈：

- 你能第一次把 `mind-gateway` 跑起来
- 已经可以录一个“服务启动成功”的 10 秒演示

## Day 2：配置、日志与 HTTP 框架定型

目标：把“能跑”提升到“有工程边界”。

工作项：

- 定义配置结构体
- 支持从本地文件读取配置
- 接入结构化日志
- 完成基于 `fasthttp` 的 router、middleware、错误响应格式
- 将统一错误响应收敛到 `handler/common.go`
- 保持资源 handler 与通用响应辅助分离

当天我会实际创建或完善的文件：

- `internal/config/config.go`
- `internal/config/load.go`
- `internal/config/load_test.go`
- `internal/app/app.go`
- `internal/transport/http/router.go`
- `internal/transport/http/server.go`
- `internal/transport/http/middleware.go`
- `internal/transport/http/handler/`
- `internal/transport/http/handler/common.go`
- `internal/transport/http/handler/health.go`
- `internal/transport/http/middleware_test.go`
- `internal/observability/request_log.go`

量化验收：

- 至少 1 个配置文件样例可被成功加载
- 日志包含 `level`、`msg`、`ts`
- 非法路径与非法 JSON 都有统一错误格式
- 形成 1 条配置加载测试
- 至少 1 条 `fasthttp` 中间件或处理器测试

当天正反馈：

- 项目不再像 demo 脚本，而开始像真正服务
- 你已经能解释“配置从哪来、请求怎么进来”

## Day 3：统一 Schema 与 Provider 接口

目标：把外部请求和内部执行世界隔开。

工作项：

- 定义 chat request / response 内部 schema
- 设计 `Provider` 接口
- 实现 `mock` provider
- 写 provider registry 初版

当天我会实际创建或完善的文件：

- `internal/core/types.go`
- `internal/provider/provider.go`
- `internal/provider/registry.go`
- `internal/provider/mock/mock.go`

量化验收：

- 至少 1 个 `Provider` 接口与 1 个实现
- `mock` provider 可返回固定 chat response
- 单元测试覆盖 schema 序列化与 provider 调用
- 测试总数达到 5 个以上

当天正反馈：

- 你已经不是在写“单一路由转发器”
- 已经可以开始说“这是一个多 provider gateway 骨架”

## Day 4：打通非流式主链路

目标：第一次端到端跑通 `/v1/chat/completions`。

工作项：

- 增加 chat completions handler
- handler 转内部 request
- 调用 gateway 初版执行链路
- 对接 `mock` provider 返回结果

当天我会实际创建或完善的文件：

- `internal/core/gateway.go`
- `internal/transport/http/handler/chat.go`
- `internal/transport/http/openai_api.go`

量化验收：

- `POST /v1/chat/completions` 返回合法 JSON
- 至少 2 条集成测试覆盖成功与参数错误
- 请求日志中可看到 `request_id`
- 能给出 1 条完整 `curl` 示例

当天正反馈：

- 第一次拿到像样的 OpenAI-compatible 返回
- 这一天之后项目已经从“服务骨架”变成“网关原型”

## Day 5：接入 OpenAI Provider

目标：让系统具备真实外部价值，不只是假数据。

工作项：

- 实现 `openai` provider
- 支持 base URL、api key、model 配置
- 完成 request / response adapter
- 增加 mock 与 openai 的切换能力

当天我会实际创建或完善的文件：

- `internal/provider/openai/client.go`
- `internal/provider/openai/adapter.go`
- `internal/provider/openai/provider.go`

量化验收：

- 至少 2 个 provider 可通过统一接口调用
- 有 1 条 openai provider 单测或 adapter 测试
- 本地可通过配置切换 provider
- README 增加真实 provider 使用说明

当天正反馈：

- 项目第一次具备真实外部联调用途
- 你已经能展示“统一接口切 provider”

## Day 6：引入 Queue + Worker 调度内核

目标：把“直接调用 provider”升级成“网关调度”。

工作项：

- 设计 `Gateway` 主对象
- 为 provider 建 queue
- 启动 worker 消费请求
- handler 改为向 gateway 提交请求

当天我会实际创建或完善的文件：

- `internal/core/queue.go`
- `internal/core/executor.go`
- `internal/core/gateway.go`

量化验收：

- 至少 1 个 provider 拥有独立 queue
- worker 数量可配置
- 压测或并发测试至少覆盖 20 个并发请求
- 日志中能看到请求入队与消费事件

当天正反馈：

- 这是第一个真正“像 Bifrost”的里程碑
- 你可以自信讲出 transport 和 core 已经分层

## Day 7：实现 Retry、Backoff、Fallback

目标：把网关做出可靠性味道。

工作项：

- 加入最大重试次数
- 加入退避策略
- 支持 fallback provider 链
- 补充错误分类与日志字段

当天我会实际创建或完善的文件：

- `internal/core/retry.go`
- `internal/core/fallback.go`
- `internal/transport/http/openai_api.go`

量化验收：

- 人为制造错误时至少触发 1 次 retry
- 主 provider 失败时能切到 fallback 成功
- 至少 3 条测试覆盖 retry、fallback、不可重试错误
- 日志出现 `retry_count` 与 `fallback_index`

当天正反馈：

- 项目从“能工作”升级为“更像生产系统”
- 你已经有足够硬的一个简历亮点

## Day 8：落 Logging Plugin

目标：把横切能力从主链路里抽出来。

工作项：

- 设计 plugin 接口
- 增加 pre-hook / post-hook
- 实现 logging plugin
- 记录输入、输出、provider、耗时、错误

当天我会实际创建或完善的文件：

- `internal/plugin/plugin.go`
- `internal/plugin/pipeline.go`
- `internal/plugin/logging/logging.go`

量化验收：

- 至少 1 个插件可注册与启停
- 1 次请求至少产出 1 条完整结构化日志
- 日志字段至少包含 `request_id`、`provider`、`latency_ms`
- 至少 2 条测试覆盖插件执行顺序

当天正反馈：

- 项目第一次具备“平台扩展点”
- 你已经能讲出“日志不是写死在 handler 里”

## Day 9：实现 Governance Lite

目标：加入一个真正有产品感的策略能力。

工作项：

- 支持 `virtual key`
- 增加额度配置
- pre-hook 做额度校验
- post-hook 做 usage 记账

当天我会实际创建或完善的文件：

- `internal/plugin/governance/governance.go`
- `internal/config/config.go`

量化验收：

- 至少支持 2 个 virtual key
- 超限请求会返回明确拒绝错误
- 正常请求会累计 usage
- 至少 3 条测试覆盖通过、拒绝、记账

当天正反馈：

- 项目正式跨过“普通代理”这条线
- 你已经可以讲 multi-tenant / budget control 的故事

## Day 10：补齐 Streaming

目标：让核心接口既支持普通响应，也支持 SSE。

工作项：

- 为 chat completions 增加 streaming 分支
- provider 支持流式返回适配
- streaming 响应保持统一事件格式
- logging plugin 适配流式完成态

当天我会实际创建或完善的文件：

- `internal/core/types.go`
- `internal/provider/openai/provider.go`
- `internal/transport/http/handler/chat.go`

量化验收：

- `stream=true` 时可以持续收到事件
- 至少 1 条集成测试覆盖 streaming
- 流结束后日志能记录总耗时与结束状态
- demo 可展示至少 5 个 chunk

当天正反馈：

- 面试展示效果明显提升
- 这是一个非常直观的“系统复杂度上台阶”节点

## Day 11：实现 Tool Registry 与 Tool Loop

目标：让 gateway 具备 agent-aware 能力。

工作项：

- 设计 tool registry
- 注册至少 2 个本地工具
- 支持识别 tool call
- 支持执行工具并回送模型继续推理

当天我会实际创建或完善的文件：

- `internal/tools/registry.go`
- `internal/tools/builtin.go`
- `internal/tools/loop.go`
- `internal/core/executor.go`

量化验收：

- 至少 2 个工具成功注册
- 至少 1 条完整 tool loop 集成测试
- 演示链路包含“模型请求工具 -> 工具执行 -> 最终回答”
- 日志能区分模型调用与工具调用

当天正反馈：

- 项目从 gateway prototype 升级为 agent-aware gateway
- 这是非常适合进简历和面试展开的一天

## Day 12：补可观测性与调试接口

目标：让请求执行路径可解释、可排查。

工作项：

- 增加 `/debug/requests` 或 `/debug/stats`
- 保存最近 N 条请求摘要
- 输出 provider、retry、fallback、latency 等字段
- 增加调试文档

当天我会实际创建或完善的文件：

- `internal/observability/debug_store.go`
- `internal/transport/http/handler/debug.go`

量化验收：

- debug 接口可查看最近请求
- 至少保留最近 20 条请求摘要
- 每条摘要包含 5 个以上关键字段
- 至少 2 条测试覆盖调试接口

当天正反馈：

- 你终于能回答“某次请求为什么慢、为什么 fallback”
- 这让项目更像 infra，而不是功能堆叠

## Day 13：测试加固与演示脚本

目标：让项目可复现、可交付、可对外展示。

工作项：

- 补齐单元测试和集成测试
- 编写 demo 脚本
- 增加样例配置与 sample requests
- 跑一次基础并发 smoke test

当天我会实际创建或完善的文件：

- `scripts/demo.sh`
- `testdata/config.dev.yaml`
- `*_test.go`

量化验收：

- `go test ./...` 主链路通过
- 自动化测试数达到 20 条以上
- demo 脚本覆盖普通请求、fallback、governance、tool loop
- 至少记录 1 轮 smoke test 结果

当天正反馈：

- 项目已经从“开发中”进入“可对外演示”
- 你能很顺手做一场 5 分钟 demo

## Day 14：收尾、文档、简历化包装

目标：把工程成果转成可讲述、可投递的作品。

工作项：

- 完成 README 正式版
- 补架构说明与执行路径图文
- 整理设计取舍与已知限制
- 输出简历表述、项目亮点、面试讲解提纲

当天我会实际创建或完善的文件：

- `README.md`
- `plans/implementation-blueprint.md`

量化验收：

- README 包含架构、启动、配置、测试、演示、限制说明
- 至少形成 3 条简历 bullet
- 至少形成 10 个常见面试问答要点
- 仓库达到“陌生人可按文档跑起来”的程度

当天正反馈：

- 你不只是做完了项目，还做完了“可展示的项目”
- 这一版已经足够支持简历、面试与后续迭代

## 里程碑检查点

第 4 天结束：

- 已有可用 API 主链路
- 可以展示最小 OpenAI-compatible 网关

第 7 天结束：

- 已有核心调度与可靠性能力
- 可以讲 queue、worker、retry、fallback

第 10 天结束：

- 已有 plugin 与 streaming
- 可以讲平台扩展性与复杂响应链路

第 12 天结束：

- 已有 governance 与 tool loop
- 可以讲 agent-aware 与策略治理

第 14 天结束：

- 已有完整作品闭环
- 可以直接进入 README 展示、简历书写与下一轮功能迭代

## 完成后的简历素材方向

- 实现 Bifrost-inspired 多 Provider AI Gateway，统一暴露 OpenAI-compatible API。
- 设计 queue + worker 调度内核，支持 retry、backoff、fallback 与结构化请求追踪。
- 构建插件化扩展机制，落地 logging、budget governance 与请求审计能力。
- 实现最小 tool loop，使网关具备本地工具执行与 agent-aware orchestration 能力。
