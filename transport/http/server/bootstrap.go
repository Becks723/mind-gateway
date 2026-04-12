package server

import (
	"fmt"

	"github.com/Becks723/mind-gateway/core"
	frameworkconfig "github.com/Becks723/mind-gateway/framework/config"
	frameworklogging "github.com/Becks723/mind-gateway/framework/logging"
	"github.com/Becks723/mind-gateway/provider"
	mockprovider "github.com/Becks723/mind-gateway/provider/mock"
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
	registry, err := buildProviderRegistry(cfg)
	if err != nil {
		return nil, err
	}

	// 创建网关核心对象
	gateway := core.NewGateway(cfg.Gateway, registry)

	return New(cfg, logger, gateway), nil
}

// buildProviderRegistry 根据配置构建 Provider 注册表
func buildProviderRegistry(cfg *frameworkconfig.Config) (*provider.Registry, error) {
	registry := provider.NewRegistry()

	for _, providerCfg := range cfg.Providers {
		if !providerCfg.Enabled {
			continue
		}

		switch providerCfg.Type {
		case "mock":
			if err := registry.Register(mockprovider.New(providerCfg.Name, providerCfg.MockResponse)); err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("暂不支持的 provider 类型: %s", providerCfg.Type)
		}
	}

	return registry, nil
}
