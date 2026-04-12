package schema

import "context"

// Provider 定义统一的模型 Provider 接口
type Provider interface {
	Name() string
	Type() string
	Chat(ctx context.Context, req *Request) (*Response, error)
	ChatStream(ctx context.Context, req *Request) (<-chan StreamEvent, <-chan error)
}
