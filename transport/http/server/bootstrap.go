package server

import (
	"fmt"
	"time"

	"github.com/Becks723/mind-gateway/core"
	"github.com/Becks723/mind-gateway/core/schema"
	frameworkconfig "github.com/Becks723/mind-gateway/framework/config"
	frameworklogging "github.com/Becks723/mind-gateway/framework/logging"
	frameworktool "github.com/Becks723/mind-gateway/framework/tool"
	"github.com/Becks723/mind-gateway/plugin"
	governanceplugin "github.com/Becks723/mind-gateway/plugin/governance"
	logplugin "github.com/Becks723/mind-gateway/plugin/logging"
	"github.com/Becks723/mind-gateway/provider"
	mockprovider "github.com/Becks723/mind-gateway/provider/mock"
	openaiprovider "github.com/Becks723/mind-gateway/provider/openai"
)

// Bootstrap 加载配置并构建 HTTP 服务
func Bootstrap(configPath string) (*Server, error) {
	// 加载应用配置
	cfg, err := frameworkconfig.Load(configPath)
	if err != nil {
		return nil, err
	}

	// 创建结构化日志记录器
	logger := frameworklogging.NewLogger(cfg.Observability.LogLevel)

	// 构建 Provider 注册表
	registry, err := buildProviderRegistry(cfg, logger)
	if err != nil {
		return nil, err
	}

	// 构建插件执行管线
	pluginPipeline := buildPluginPipeline(cfg, logger)

	// 构建工具注册表
	toolRegistry, err := buildToolRegistry(cfg)
	if err != nil {
		return nil, err
	}

	// 创建网关核心对象
	gateway := core.NewGateway(cfg.Gateway, registry, logger, pluginPipeline, toolRegistry, cfg.Providers)

	return New(cfg, logger, gateway), nil
}

// buildToolRegistry 根据配置构建工具注册表
func buildToolRegistry(cfg *frameworkconfig.Config) (*frameworktool.Registry, error) {
	// 在工具能力关闭时跳过注册
	if cfg == nil || !cfg.Tools.Enabled {
		return nil, nil
	}

	registry := frameworktool.NewRegistry()
	if err := frameworktool.RegisterBuiltinTools(registry, cfg.Tools.AllowedTools); err != nil {
		return nil, err
	}

	return registry, nil
}

// buildProviderRegistry 根据配置构建 Provider 注册表
func buildProviderRegistry(cfg *frameworkconfig.Config, logger *frameworklogging.Logger) (*provider.Registry, error) {
	registry := provider.NewRegistry()

	for _, providerCfg := range cfg.Providers {
		// 跳过未启用的 Provider
		if !providerCfg.Enabled {
			continue
		}

		// 根据配置创建具体 Provider
		instance, err := buildProvider(providerCfg, cfg.Gateway.RequestTimeout)
		if err != nil {
			return nil, err
		}

		// 注册 Provider 并输出日志
		if err := registry.Register(instance); err != nil {
			return nil, err
		}
		logger.Info("注册 provider 成功", "provider", providerCfg.Name, "type", providerCfg.Type)
	}

	return registry, nil
}

// buildProvider 根据配置创建具体 Provider
func buildProvider(providerCfg frameworkconfig.ProviderConfig, requestTimeout time.Duration) (schema.Provider, error) {
	// 按类型分发到具体的 Provider 构造函数
	switch providerCfg.Type {
	case "mock":
		return mockprovider.New(providerCfg.Name, providerCfg.MockResponse), nil
	case "openai":
		return openaiprovider.NewProvider(
			providerCfg.Name,
			providerCfg.BaseURL,
			providerCfg.APIKey,
			providerCfg.ModelMap,
			requestTimeout,
		), nil
	default:
		return nil, fmt.Errorf("暂不支持的 provider 类型: %s", providerCfg.Type)
	}
}

// buildPluginPipeline 根据配置构建插件执行管线
func buildPluginPipeline(cfg *frameworkconfig.Config, logger *frameworklogging.Logger) *plugin.Pipeline {
	// 收集全部启用的插件
	plugins := make([]schema.Plugin, 0, 2)
	if cfg.Plugins.LoggingEnabled {
		plugins = append(plugins, logplugin.NewPlugin(logger))
	}
	if cfg.Plugins.GovernanceEnabled && cfg.Governance.Enabled {
		plugins = append(plugins, governanceplugin.NewPlugin(logger, cfg.Governance))
	}

	return plugin.NewPipeline(plugins...)
}
