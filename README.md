# mind-gateway

`mind-gateway` 是一个面向多模型接入场景的 AI gateway 项目。

它提供统一的 AI gateway 形态，围绕 OpenAI-compatible 接口、多 provider 抽象、请求调度、可靠性能力、插件扩展和工具执行能力进行实现。

## 目录结构

```text
cmd/server/main.go
internal/app/app.go
internal/transport/http/
internal/transport/http/handler/
plans/
testdata/
```

## 本地启动

```bash
go run ./cmd/server
```

服务默认监听 `127.0.0.1:8080`。

## 健康检查

```bash
curl http://127.0.0.1:8080/healthz
```

预期返回：

```json
{
  "status": "ok",
  "service": "mind-gateway",
  "timestamp": "2026-04-12T00:00:00Z"
}
```

## 项目目标

- 提供 OpenAI-compatible API 入口
- 统一多 provider 调用方式
- 支持 queue、worker、retry、fallback 等网关能力
- 提供 plugin、governance 与 tool loop 扩展能力

## 设计文档

- [plans/requirements.md](plans/requirements.md)
- [plans/implementation-blueprint.md](plans/implementation-blueprint.md)
- [plans/daily-plan.md](plans/daily-plan.md)
- [plans/review-workflow.md](plans/review-workflow.md)
- [plans/review-log.md](plans/review-log.md)
- [CODE_STYLE.md](CODE_STYLE.md)
