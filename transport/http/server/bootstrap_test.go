package server

import "testing"

// TestBootstrap 验证可以基于测试配置构建 HTTP 服务
func TestBootstrap(t *testing.T) {
	server, err := Bootstrap("../../../testdata/config.dev.yaml")
	if err != nil {
		t.Fatalf("构建 HTTP 服务失败: %v", err)
	}

	if server == nil {
		t.Fatal("期望返回非空服务对象")
	}
	if server.Addr == "" {
		t.Fatal("期望服务监听地址非空")
	}
}
