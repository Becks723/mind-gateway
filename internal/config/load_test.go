package config

import "testing"

// TestLoad 验证可以从默认测试配置中成功加载配置
func TestLoad(t *testing.T) {
	// 加载默认测试配置
	cfg, err := Load("../../testdata/config.dev.yaml")
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	// 校验关键配置字段
	if cfg.Server.Host != "127.0.0.1" {
		t.Fatalf("期望 Host 为 127.0.0.1，实际得到 %q", cfg.Server.Host)
	}
	if cfg.Server.Port != 8080 {
		t.Fatalf("期望 Port 为 8080，实际得到 %d", cfg.Server.Port)
	}
	if len(cfg.Providers) != 1 {
		t.Fatalf("期望 Provider 数量为 1，实际得到 %d", len(cfg.Providers))
	}
}

// TestLoadWithDefaultPath 验证默认值逻辑可以补齐缺失配置
func TestLoadWithDefaultPath(t *testing.T) {
	// 构造最小配置并应用默认值
	cfg := &Config{}
	applyDefaults(cfg)

	// 校验默认值
	if cfg.Server.Host != "127.0.0.1" {
		t.Fatalf("期望默认 Host 为 127.0.0.1，实际得到 %q", cfg.Server.Host)
	}
	if cfg.Server.Port != 8080 {
		t.Fatalf("期望默认 Port 为 8080，实际得到 %d", cfg.Server.Port)
	}
	if cfg.Observability.LogLevel != "info" {
		t.Fatalf("期望默认日志级别为 info，实际得到 %q", cfg.Observability.LogLevel)
	}
}
