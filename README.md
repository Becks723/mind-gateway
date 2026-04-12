# mind-gateway

`mind-gateway` 是一个面向多模型接入场景的 AI gateway 项目。

它提供统一的 AI gateway 形态，围绕 OpenAI-compatible 接口、多 provider 抽象、请求调度、可靠性能力、插件扩展和工具执行能力进行实现。

## 目录结构

```text
cmd/mind_http/main.go
core/
framework/
plugin/
provider/
transport/http/
plans/
testdata/
```

## 本地启动

```bash
go run ./cmd/mind_http
```

服务默认监听 `127.0.0.1:8080`。

可以通过环境变量指定配置文件路径：

```bash
MIND_GATEWAY_CONFIG=testdata/config.dev.yaml go run ./cmd/mind_http
```

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

## Chat Completions

```bash
curl -X POST http://127.0.0.1:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "mock-gpt",
    "messages": [
      {"role": "user", "content": "你好"}
    ]
  }'
```

响应格式兼容 OpenAI 风格的 `chat.completion` 结构。

## 项目目标

- 提供 OpenAI-compatible API 入口
- 统一多 provider 调用方式
- 支持 queue、worker、retry、fallback 等网关能力
- 提供 plugin、governance 与 tool loop 扩展能力

## 基础能力

- `fasthttp` 作为 HTTP 传输层
- YAML 配置加载
- 中文结构化请求日志
- 统一 JSON 错误响应
- `/healthz`
- `/v1/chat/completions`
- `mock` provider
- Provider registry
- 最小 gateway 执行入口
- 自动化测试

## 配置文件

默认配置文件示例位于：

```text
testdata/config.dev.yaml
```

其中包括：

- 服务监听地址与超时
- 默认 provider 与模型
- mock provider 配置
- 日志等级
- 调试缓冲区大小

