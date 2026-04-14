package governance

import (
	"context"
	"testing"

	"github.com/Becks723/mind-gateway/core/schema"
	frameworkconfig "github.com/Becks723/mind-gateway/framework/config"
	frameworklogging "github.com/Becks723/mind-gateway/framework/logging"
	"github.com/valyala/fasthttp"
)

// TestPluginPreHookAllow 验证合法 virtual key 可以通过治理校验
func TestPluginPreHookAllow(t *testing.T) {
	plugin := NewPlugin(frameworklogging.NewLogger("error"), frameworkconfig.GovernanceConfig{
		VirtualKeys: []frameworkconfig.VirtualKeyConfig{
			{
				Key:              "vk-alpha",
				Name:             "alpha",
				AllowedProviders: []string{"mock"},
				AllowedModels:    []string{"mock-gpt"},
			},
		},
	})

	req := &schema.Request{
		RequestID:  "req-1",
		Provider:   "mock",
		Model:      "mock-gpt",
		VirtualKey: "vk-alpha",
	}

	result, err := plugin.PreHook(context.Background(), req)
	if err != nil {
		t.Fatalf("期望治理校验通过，实际失败: %v", err)
	}
	if result != nil {
		t.Fatalf("期望不返回短路结果，实际得到 %#v", result)
	}
	if req.Metadata["virtual_key_name"] != "alpha" {
		t.Fatalf("期望写入 virtual_key_name=alpha，实际得到 %#v", req.Metadata["virtual_key_name"])
	}
}

// TestPluginPreHookRejectByQuota 验证超过请求额度会被拒绝
func TestPluginPreHookRejectByQuota(t *testing.T) {
	plugin := NewPlugin(frameworklogging.NewLogger("error"), frameworkconfig.GovernanceConfig{
		VirtualKeys: []frameworkconfig.VirtualKeyConfig{
			{
				Key:         "vk-beta",
				Name:        "beta",
				MaxRequests: 1,
			},
		},
	})

	firstReq := &schema.Request{
		RequestID:  "req-2",
		Provider:   "mock",
		Model:      "mock-gpt",
		VirtualKey: "vk-beta",
	}
	if _, err := plugin.PreHook(context.Background(), firstReq); err != nil {
		t.Fatalf("期望首次请求通过治理校验，实际失败: %v", err)
	}
	if _, err := plugin.PostHook(context.Background(), firstReq, &schema.Response{
		Usage: schema.Usage{
			InputTokens:  1,
			OutputTokens: 1,
		},
	}, nil); err != nil {
		t.Fatalf("期望首次请求记账成功，实际失败: %v", err)
	}

	_, err := plugin.PreHook(context.Background(), &schema.Request{
		RequestID:  "req-3",
		Provider:   "mock",
		Model:      "mock-gpt",
		VirtualKey: "vk-beta",
	})
	if err == nil {
		t.Fatal("期望第二次请求因额度超限被拒绝")
	}

	rejectionErr, ok := err.(*RejectionError)
	if !ok {
		t.Fatalf("期望返回 RejectionError，实际得到 %T", err)
	}
	if rejectionErr.StatusCode() != fasthttp.StatusTooManyRequests {
		t.Fatalf("期望状态码为 429，实际得到 %d", rejectionErr.StatusCode())
	}
}

// TestPluginPostHookAccounting 验证成功请求会累计 usage
func TestPluginPostHookAccounting(t *testing.T) {
	plugin := NewPlugin(frameworklogging.NewLogger("error"), frameworkconfig.GovernanceConfig{
		VirtualKeys: []frameworkconfig.VirtualKeyConfig{
			{
				Key:  "vk-gamma",
				Name: "gamma",
			},
		},
	})

	req := &schema.Request{
		RequestID:  "req-4",
		Provider:   "mock",
		Model:      "mock-gpt",
		VirtualKey: "vk-gamma",
	}
	resp := &schema.Response{
		Usage: schema.Usage{
			InputTokens:  11,
			OutputTokens: 7,
		},
	}

	if _, err := plugin.PostHook(context.Background(), req, resp, nil); err != nil {
		t.Fatalf("期望记账成功，实际失败: %v", err)
	}

	usage, ok := plugin.CurrentUsage("vk-gamma")
	if !ok {
		t.Fatal("期望读取到虚拟密钥使用量")
	}
	if usage.RequestCount != 1 {
		t.Fatalf("期望请求数为 1，实际得到 %d", usage.RequestCount)
	}
	if usage.InputTokens != 11 {
		t.Fatalf("期望输入 token 为 11，实际得到 %d", usage.InputTokens)
	}
	if usage.OutputTokens != 7 {
		t.Fatalf("期望输出 token 为 7，实际得到 %d", usage.OutputTokens)
	}
}

// TestPluginSupportsMultipleVirtualKeys 验证多把 virtual key 的使用量相互隔离
func TestPluginSupportsMultipleVirtualKeys(t *testing.T) {
	plugin := NewPlugin(frameworklogging.NewLogger("error"), frameworkconfig.GovernanceConfig{
		VirtualKeys: []frameworkconfig.VirtualKeyConfig{
			{
				Key:  "vk-a",
				Name: "tenant-a",
			},
			{
				Key:  "vk-b",
				Name: "tenant-b",
			},
		},
	})

	if _, err := plugin.PostHook(context.Background(), &schema.Request{
		RequestID:  "req-a",
		Provider:   "mock",
		Model:      "mock-gpt",
		VirtualKey: "vk-a",
	}, &schema.Response{
		Usage: schema.Usage{
			InputTokens:  3,
			OutputTokens: 2,
		},
	}, nil); err != nil {
		t.Fatalf("记录 tenant-a 使用量失败: %v", err)
	}

	firstUsage, ok := plugin.CurrentUsage("vk-a")
	if !ok {
		t.Fatal("期望读取到 tenant-a 的使用量")
	}
	secondUsage, ok := plugin.CurrentUsage("vk-b")
	if !ok {
		t.Fatal("期望读取到 tenant-b 的使用量")
	}

	if firstUsage.RequestCount != 1 || firstUsage.InputTokens != 3 || firstUsage.OutputTokens != 2 {
		t.Fatalf("tenant-a 的使用量不符合预期: %#v", firstUsage)
	}
	if secondUsage.RequestCount != 0 || secondUsage.InputTokens != 0 || secondUsage.OutputTokens != 0 {
		t.Fatalf("tenant-b 的使用量应保持初始值，实际得到 %#v", secondUsage)
	}
}
