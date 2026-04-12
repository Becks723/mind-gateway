package provider

import (
	"context"

	"github.com/Becks723/mind-gateway/internal/core"
)

// Provider 定义统一的模型 Provider 接口
type Provider interface {
	Name() string
	Type() string
	Chat(ctx context.Context, req *core.Request) (*core.Response, error)
	ChatStream(ctx context.Context, req *core.Request) (<-chan core.StreamEvent, <-chan error)
}
