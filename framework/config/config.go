package config

import "time"

// Config 定义应用的根配置结构
type Config struct {
	Server        ServerConfig        `yaml:"server"`        // Server 表示 HTTP 服务配置
	Gateway       GatewayConfig       `yaml:"gateway"`       // Gateway 表示网关核心配置
	Providers     []ProviderConfig    `yaml:"providers"`     // Providers 表示 Provider 列表配置
	Plugins       PluginsConfig       `yaml:"plugins"`       // Plugins 表示插件开关配置
	Governance    GovernanceConfig    `yaml:"governance"`    // Governance 表示治理配置
	Tools         ToolsConfig         `yaml:"tools"`         // Tools 表示工具配置
	Observability ObservabilityConfig `yaml:"observability"` // Observability 表示观测配置
}

// ServerConfig 定义 HTTP 服务启动配置
type ServerConfig struct {
	Host            string        `yaml:"host"`             // Host 表示服务监听地址
	Port            int           `yaml:"port"`             // Port 表示服务监听端口
	ReadTimeout     time.Duration `yaml:"read_timeout"`     // ReadTimeout 表示请求读取超时
	WriteTimeout    time.Duration `yaml:"write_timeout"`    // WriteTimeout 表示响应写入超时
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout"` // ShutdownTimeout 表示优雅关闭超时
}

// GatewayConfig 定义网关核心配置
type GatewayConfig struct {
	DefaultProvider    string        `yaml:"default_provider"`     // DefaultProvider 表示默认 Provider 名称
	DefaultModel       string        `yaml:"default_model"`        // DefaultModel 表示默认模型名称
	RequestTimeout     time.Duration `yaml:"request_timeout"`      // RequestTimeout 表示单次请求超时
	MaxRetries         int           `yaml:"max_retries"`          // MaxRetries 表示最大重试次数
	RetryBackoff       time.Duration `yaml:"retry_backoff"`        // RetryBackoff 表示初始重试退避时长
	MaxBackoff         time.Duration `yaml:"max_backoff"`          // MaxBackoff 表示最大退避时长
	QueueSize          int           `yaml:"queue_size"`           // QueueSize 表示 Provider 队列长度
	WorkersPerProvider int           `yaml:"workers_per_provider"` // WorkersPerProvider 表示每个 Provider 的工作协程数
	DropOnQueueFull    bool          `yaml:"drop_on_queue_full"`   // DropOnQueueFull 表示队列满时是否直接丢弃
	DebugBufferSize    int           `yaml:"debug_buffer_size"`    // DebugBufferSize 表示调试缓冲区容量
}

// ProviderConfig 定义单个 Provider 配置
type ProviderConfig struct {
	Name            string            `yaml:"name"`             // Name 表示 Provider 名称
	Type            string            `yaml:"type"`             // Type 表示 Provider 类型
	BaseURL         string            `yaml:"base_url"`         // BaseURL 表示 Provider 基础地址
	APIKey          string            `yaml:"api_key"`          // APIKey 表示 Provider 访问密钥
	ModelMap        map[string]string `yaml:"model_map"`        // ModelMap 表示模型映射关系
	Fallbacks       []string          `yaml:"fallbacks"`        // Fallbacks 表示降级 Provider 列表
	Enabled         bool              `yaml:"enabled"`          // Enabled 表示当前 Provider 是否启用
	MockResponse    string            `yaml:"mock_response"`    // MockResponse 表示 mock Provider 的固定返回内容
	SimulateFailure bool              `yaml:"simulate_failure"` // SimulateFailure 表示是否模拟失败
}

// PluginsConfig 定义插件开关配置
type PluginsConfig struct {
	LoggingEnabled    bool `yaml:"logging_enabled"`    // LoggingEnabled 表示是否启用日志插件
	GovernanceEnabled bool `yaml:"governance_enabled"` // GovernanceEnabled 表示是否启用治理插件
}

// GovernanceConfig 定义治理配置
type GovernanceConfig struct {
	Enabled     bool               `yaml:"enabled"`      // Enabled 表示是否启用治理能力
	VirtualKeys []VirtualKeyConfig `yaml:"virtual_keys"` // VirtualKeys 表示虚拟密钥列表
}

// VirtualKeyConfig 定义单个虚拟密钥配置
type VirtualKeyConfig struct {
	Key              string   `yaml:"key"`               // Key 表示虚拟密钥值
	Name             string   `yaml:"name"`              // Name 表示虚拟密钥名称
	MaxRequests      int64    `yaml:"max_requests"`      // MaxRequests 表示最大请求次数
	MaxInputTokens   int64    `yaml:"max_input_tokens"`  // MaxInputTokens 表示最大输入 Token 数
	MaxOutputTokens  int64    `yaml:"max_output_tokens"` // MaxOutputTokens 表示最大输出 Token 数
	AllowedProviders []string `yaml:"allowed_providers"` // AllowedProviders 表示允许的 Provider 列表
	AllowedModels    []string `yaml:"allowed_models"`    // AllowedModels 表示允许的模型列表
}

// ToolsConfig 定义工具配置
type ToolsConfig struct {
	Enabled      bool     `yaml:"enabled"`       // Enabled 表示是否启用工具能力
	AllowedTools []string `yaml:"allowed_tools"` // AllowedTools 表示允许的工具列表
}

// ObservabilityConfig 定义观测配置
type ObservabilityConfig struct {
	LogLevel           string `yaml:"log_level"`            // LogLevel 表示日志等级
	KeepRecentRequests int    `yaml:"keep_recent_requests"` // KeepRecentRequests 表示保留的近期请求数量
}
