package provider

import (
	"context"

	"github.com/Becks723/mind-gateway/core/schema"
)

// Provider 定义统一的模型 Provider 接口
type Provider interface {
	Name() string
	Type() string
	Chat(ctx context.Context, req *schema.Request) (*schema.Response, error)
	ChatStream(ctx context.Context, req *schema.Request) (<-chan schema.StreamEvent, <-chan error)
}
