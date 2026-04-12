package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

const defaultConfigPath = "testdata/config.dev.yaml"

// Load 从指定路径加载应用配置
func Load(path string) (*Config, error) {
	// 计算最终配置路径
	finalPath := path
	if finalPath == "" {
		finalPath = defaultConfigPath
	}

	// 读取配置文件内容
	data, err := os.ReadFile(finalPath)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	// 解析 YAML 配置
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 补齐必要的默认值
	applyDefaults(&cfg)

	return &cfg, nil
}

// applyDefaults 为缺失配置补齐默认值
func applyDefaults(cfg *Config) {
	// 补齐服务默认配置
	if cfg.Server.Host == "" {
		cfg.Server.Host = "127.0.0.1"
	}
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}
	if cfg.Observability.LogLevel == "" {
		cfg.Observability.LogLevel = "info"
	}
	if cfg.Observability.KeepRecentRequests == 0 {
		cfg.Observability.KeepRecentRequests = 20
	}
}
