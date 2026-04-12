# mind-gateway 实现蓝图

这份文档不是讨论稿，而是后续编码阶段默认遵循的实现合同。  
我的职责是按这份蓝图独立落地，而不是把设计继续抛回给你。

编码实现时，额外遵循根目录的 `CODE_STYLE.md`。
每完成一个 Day 后，额外遵循 `plans/review-workflow.md` 中定义的审阅、记录与计划回写流程。
如果遇到不确定或不会写的实现，优先直接参考 `bifrost/` 下的源码，不自行发散设计。
目录与文件命名默认遵循“单数优先 + snake_case”规则。

## 1. 顶层目标

我将实现一个 `Bifrost-inspired` 的精简 AI gateway，具备：

- OpenAI-compatible chat completions API
- 多 provider 统一抽象
- provider 级 queue + worker 调度
- retry / backoff / fallback
- plugin pipeline
- governance lite
- tool loop
- debug / observability 基础能力

传输层框架固定为 `fasthttp`，不再使用原生 `net/http` 作为主服务框架。

## 2. 冻结目录结构

```text
mind-gateway/
├── cmd/server/main.go
├── internal/app/
│   └── app.go
├── internal/config/
│   ├── config.go
│   └── load.go
├── internal/core/
│   ├── executor.go
│   ├── fallback.go
│   ├── gateway.go
│   ├── queue.go
│   ├── retry.go
│   └── types.go
├── internal/provider/
│   ├── provider.go
│   ├── registry.go
│   ├── mock/
│   │   └── mock.go
│   └── openai/
│       ├── adapter.go
│       ├── client.go
│       └── provider.go
├── internal/plugin/
│   ├── pipeline.go
│   ├── plugin.go
│   ├── governance/
│   │   └── governance.go
│   └── logging/
│       └── logging.go
├── internal/tools/
│   ├── builtin.go
│   ├── loop.go
│   └── registry.go
├── internal/transport/http/
│   ├── handler/
│   │   ├── chat.go
│   │   ├── debug.go
│   │   └── health.go
│   ├── middleware.go
│   ├── openai_api.go
│   ├── router.go
│   └── server.go
├── internal/observability/
│   ├── debug_store.go
│   └── request_log.go
├── scripts/demo.sh
├── testdata/config.dev.yaml
├── plans/
└── README.md
```

## 3. 固定路由

### 3.1 第一阶段固定路由

- `GET /healthz`
- `POST /v1/chat/completions`

### 3.2 第二阶段固定路由

- `GET /debug/requests`

不新增额外公开 API，除非它直接服务于既定里程碑。

## 4. 固定配置模型

配置根对象：

```go
type Config struct {
    Server        ServerConfig        `yaml:"server"`
    Gateway       GatewayConfig       `yaml:"gateway"`
    Providers     []ProviderConfig    `yaml:"providers"`
    Plugins       PluginsConfig       `yaml:"plugins"`
    Governance    GovernanceConfig    `yaml:"governance"`
    Tools         ToolsConfig         `yaml:"tools"`
    Observability ObservabilityConfig `yaml:"observability"`
}
```

### 4.1 ServerConfig

```go
type ServerConfig struct {
    Host            string        `yaml:"host"`
    Port            int           `yaml:"port"`
    ReadTimeout     time.Duration `yaml:"read_timeout"`
    WriteTimeout    time.Duration `yaml:"write_timeout"`
    ShutdownTimeout time.Duration `yaml:"shutdown_timeout"`
}
```

### 4.2 GatewayConfig

```go
type GatewayConfig struct {
    DefaultProvider   string        `yaml:"default_provider"`
    DefaultModel      string        `yaml:"default_model"`
    RequestTimeout    time.Duration `yaml:"request_timeout"`
    MaxRetries        int           `yaml:"max_retries"`
    RetryBackoff      time.Duration `yaml:"retry_backoff"`
    MaxBackoff        time.Duration `yaml:"max_backoff"`
    QueueSize         int           `yaml:"queue_size"`
    WorkersPerProvider int          `yaml:"workers_per_provider"`
    DropOnQueueFull   bool          `yaml:"drop_on_queue_full"`
    DebugBufferSize   int           `yaml:"debug_buffer_size"`
}
```

### 4.3 ProviderConfig

```go
type ProviderConfig struct {
    Name              string   `yaml:"name"`
    Type              string   `yaml:"type"`
    BaseURL           string   `yaml:"base_url"`
    APIKey            string   `yaml:"api_key"`
    ModelMap          map[string]string `yaml:"model_map"`
    Fallbacks         []string `yaml:"fallbacks"`
    Enabled           bool     `yaml:"enabled"`
    MockResponse      string   `yaml:"mock_response"`
    SimulateFailure   bool     `yaml:"simulate_failure"`
}
```

### 4.4 PluginsConfig

```go
type PluginsConfig struct {
    LoggingEnabled    bool `yaml:"logging_enabled"`
    GovernanceEnabled bool `yaml:"governance_enabled"`
}
```

### 4.5 GovernanceConfig

```go
type GovernanceConfig struct {
    Enabled     bool                        `yaml:"enabled"`
    VirtualKeys []VirtualKeyConfig          `yaml:"virtual_keys"`
}

type VirtualKeyConfig struct {
    Key                 string `yaml:"key"`
    Name                string `yaml:"name"`
    MaxRequests         int64  `yaml:"max_requests"`
    MaxInputTokens      int64  `yaml:"max_input_tokens"`
    MaxOutputTokens     int64  `yaml:"max_output_tokens"`
    AllowedProviders    []string `yaml:"allowed_providers"`
    AllowedModels       []string `yaml:"allowed_models"`
}
```

### 4.6 ToolsConfig

```go
type ToolsConfig struct {
    Enabled          bool     `yaml:"enabled"`
    AllowedTools     []string `yaml:"allowed_tools"`
}
```

### 4.7 ObservabilityConfig

```go
type ObservabilityConfig struct {
    LogLevel         string `yaml:"log_level"`
    KeepRecentRequests int  `yaml:"keep_recent_requests"`
}
```

## 5. 固定 OpenAI-compatible API 结构

### 5.1 外部请求结构

`internal/transport/http/openai_api.go` 中固定定义：

```go
type ChatCompletionRequest struct {
    Model       string         `json:"model"`
    Messages    []ChatMessage  `json:"messages"`
    Stream      bool           `json:"stream,omitempty"`
    Temperature *float64       `json:"temperature,omitempty"`
    MaxTokens   *int           `json:"max_tokens,omitempty"`
    Tools       []ToolSpec     `json:"tools,omitempty"`
    ToolChoice  any            `json:"tool_choice,omitempty"`
    User        string         `json:"user,omitempty"`
    Metadata    map[string]any `json:"metadata,omitempty"`
}
```

```go
type ChatMessage struct {
    Role       string        `json:"role"`
    Content    any           `json:"content"`
    ToolCalls  []ToolCall    `json:"tool_calls,omitempty"`
    ToolCallID string        `json:"tool_call_id,omitempty"`
    Name       string        `json:"name,omitempty"`
}
```

### 5.2 内部核心请求结构

`internal/core/types.go` 中固定定义：

```go
type Request struct {
    RequestID       string
    Provider        string
    Model           string
    Messages        []Message
    Stream          bool
    Temperature     *float64
    MaxTokens       *int
    Tools           []ToolDefinition
    Metadata        map[string]any
    VirtualKey      string
    FallbackIndex   int
    RetryCount      int
    StartedAt       time.Time
}
```

```go
type Response struct {
    RequestID      string
    Provider       string
    Model          string
    OutputText     string
    Messages       []Message
    FinishReason   string
    Usage          Usage
    ToolCalls      []ToolInvocation
    Latency        time.Duration
}
```

### 5.3 Usage

```go
type Usage struct {
    InputTokens  int64
    OutputTokens int64
    TotalTokens  int64
}
```

## 6. 固定 Provider 接口

`internal/provider/provider.go`

```go
type Provider interface {
    Name() string
    Type() string
    Chat(ctx context.Context, req *core.Request) (*core.Response, error)
    ChatStream(ctx context.Context, req *core.Request) (<-chan core.StreamEvent, <-chan error)
}
```

不再额外定义 embeddings、images、responses 等接口。

## 7. 固定 Gateway 结构

`internal/core/gateway.go`

```go
type Gateway struct {
    cfg         config.GatewayConfig
    registry    *provider.Registry
    pipeline    *plugin.Pipeline
    tools       *tools.Registry
    debugStore  *observability.DebugStore
    queues      map[string]*ProviderQueue
    cancel      context.CancelFunc
    mu          sync.RWMutex
}
```

### 7.1 ProviderQueue

```go
type ProviderQueue struct {
    Name    string
    Queue   chan *WorkItem
    Closing atomic.Bool
    WG      sync.WaitGroup
}
```

### 7.2 WorkItem

```go
type WorkItem struct {
    Ctx      context.Context
    Request  *Request
    Response chan *Response
    Stream   chan StreamEvent
    Err      chan error
}
```

## 8. 固定插件接口

`internal/plugin/plugin.go`

```go
type Plugin interface {
    Name() string
    PreHook(ctx context.Context, req *core.Request) (*PreHookResult, error)
    PostHook(ctx context.Context, req *core.Request, resp *core.Response, runErr error) error
}
```

```go
type PreHookResult struct {
    ShortCircuit bool
    Response     *core.Response
}
```

### 8.1 Logging Plugin 职责冻结

- 在 `PreHook` 记录请求开始信息
- 在 `PostHook` 记录结果、错误、provider、latency、retry、fallback
- 输出 JSON 结构化日志

### 8.2 Governance Plugin 职责冻结

- 从 `Authorization: Bearer <virtual_key>` 或 `X-Virtual-Key` 读取 virtual key
- 在 `PreHook` 做 key 存在校验、provider/model 白名单校验、配额校验
- 在 `PostHook` 累加 request 与 token usage

## 9. 固定 Tool 接口

`internal/tools/registry.go`

```go
type Tool interface {
    Name() string
    Description() string
    InputSchema() map[string]any
    Execute(ctx context.Context, input json.RawMessage) (string, error)
}
```

第一版内置工具固定为：

- `echo`
- `time_now`

## 10. 固定执行链路

非流式请求固定执行顺序：

1. HTTP handler 解析 OpenAI-compatible request
2. 转为 `core.Request`
3. 补齐 `request_id`、`provider`、`model`
4. 执行 plugin `PreHook`
5. 若 short-circuit，直接返回
6. 将请求投递到对应 provider queue
7. worker 执行 provider call
8. provider 出错时做 retry
9. retry 失败时按 fallback 链继续
10. 若响应带 tool call，则执行 tool loop
11. 写入 debug store
12. 执行 plugin `PostHook`
13. 返回 OpenAI-compatible response

流式请求固定执行顺序：

1. 前 6 步一致
2. worker 调 provider stream
3. HTTP handler 以 SSE 输出 chunk
4. 流结束后写 debug store
5. 执行 plugin `PostHook`

## 11. 固定错误模型

统一错误返回：

```go
type APIError struct {
    Error ErrorBody `json:"error"`
}

type ErrorBody struct {
    Message string `json:"message"`
    Type    string `json:"type"`
    Code    string `json:"code,omitempty"`
}
```

预置错误类型：

- `invalid_request_error`
- `provider_error`
- `queue_overflow`
- `governance_error`
- `internal_error`

## 12. 固定测试策略

### 12.1 单元测试

- config load
- request validation
- mock provider
- openai adapter
- retry/fallback
- plugin pipeline
- governance budget
- tool registry

### 12.2 集成测试

- healthz
- chat completion happy path
- invalid request
- fallback path
- governance reject
- streaming
- tool loop

### 12.3 完成阈值

- 总测试数不少于 20
- 核心主链路必须有自动化测试
- 每个高级能力至少 1 条测试

## 13. 固定每日产出方式

后续编码阶段，我将按天完成并提交这些类型的产出：

- 新增/更新源码文件
- 测试文件
- README 或 demo 文档
- 一条可运行验证命令

不会只停留在“下一步建议做什么”。

## 14. 完成定义

以下条件全部满足才视为 `mind-gateway` 第一版落地完成：

1. 服务可启动。
2. 非流式请求可跑通。
3. 流式请求可跑通。
4. `mock` 与 `openai` provider 均可用。
5. retry / fallback / governance / tool loop 都可演示。
6. `go test ./...` 主链路通过。
7. README 足够让陌生人跑通 demo。
