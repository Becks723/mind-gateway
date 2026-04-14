# mind-gateway

`mind-gateway` 是一个面向多模型接入场景的 AI gateway 项目。

它提供统一的 AI gateway 形态，围绕 OpenAI-compatible 接口、多 provider 抽象、请求调度、可靠性能力、插件扩展、治理能力和工具执行能力进行实现。

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

主要分层职责：

- `core/`：请求 schema、Gateway、queue、worker、retry、fallback
- `framework/`：配置、日志、调试等基础设施
- `plugin/`：logging、governance 等横切插件
- `provider/`：`mock`、`openai` 等模型厂商适配
- `transport/http/`：HTTP 路由、middleware、handler、server 装配

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
  -H "Authorization: Bearer vk-dev-alpha" \
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
- 支持基于 `virtual key` 的多租户治理与额度控制

## 基础能力

- `fasthttp` 作为 HTTP 传输层
- YAML 配置加载
- 中文结构化请求日志
- 统一 JSON 错误响应
- `/healthz`
- `/v1/chat/completions`
- `mock` provider
- `openai` provider
- Provider registry
- queue + worker 调度
- retry + fallback
- logging plugin
- governance plugin
- `virtual key` 校验与 usage 记账
- 自动化测试

## 配置文件

默认配置文件示例位于：

```text
testdata/config.dev.yaml
```

其中包括：

- 服务监听地址与超时
- 默认 provider 与模型
- provider 配置
- 插件开关
- `virtual key` 与治理额度配置
- 日志等级
- 调试缓冲区大小

## Mock Provider 配置示例

```yaml
gateway:
  default_provider: mock
  default_model: mock-gpt

providers:
  - name: mock
    type: mock
    enabled: true
    mock_response: "hello from mock provider"

plugins:
  logging_enabled: true
  governance_enabled: true

governance:
  enabled: true
  virtual_keys:
    - key: vk-dev-alpha
      name: dev-alpha
      max_requests: 100
      allowed_providers:
        - mock
      allowed_models:
        - mock-gpt
```

## OpenAI Provider 配置示例

```yaml
gateway:
  default_provider: openai
  default_model: gpt-4o-mini

providers:
  - name: openai
    type: openai
    enabled: true
    base_url: https://api.openai.com
    api_key: ${OPENAI_API_KEY}
    model_map:
      gpt-4o-mini: gpt-4o-mini
```

`model_map` 的含义是：

- 键：网关对外暴露或内部使用的逻辑模型名
- 值：实际发送给底层 provider 的模型名

这可以用来做模型别名映射，也可以兼容不同厂商的模型命名差异。

启动时指定配置文件：

```bash
MIND_GATEWAY_CONFIG=testdata/config.openai.yaml go run ./cmd/mind_http
```

## Virtual Key 用法

`mind-gateway` 支持通过 `virtual key` 做最小治理能力。

支持两种传递方式：

1. `Authorization: Bearer <virtual_key>`
2. `X-Mind-Virtual-Key: <virtual_key>`

示例：

```bash
curl -X POST http://127.0.0.1:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer vk-dev-alpha" \
  -d '{
    "model": "mock-gpt",
    "messages": [
      {"role": "user", "content": "介绍一下你自己"}
    ]
  }'
```

当治理插件启用后，网关会在请求进入核心执行链路前完成：

- `virtual key` 是否存在
- Provider 是否允许
- 模型是否允许
- 请求次数与 Token 额度是否超限

请求成功后会累计：

- 成功请求数
- 输入 Token 数
- 输出 Token 数

## 错误响应

HTTP 错误统一返回：

```json
{
  "error": {
    "message": "virtual key 请求次数已超限",
    "type": "rate_limit_error",
    "code": "request_quota_exceeded"
  }
}
```

其中：

- `message`：面向人的错误说明
- `type`：稳定的错误分类
- `code`：更细粒度的机器可读错误码
