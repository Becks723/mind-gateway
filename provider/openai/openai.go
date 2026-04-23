package openai

import (
	"context"
	"fmt"
	"time"

	"github.com/Becks723/mind-gateway/core/schema"
)

// Provider 表示 OpenAI Provider 实现
type Provider struct {
	name     string            // name 表示 Provider 名称
	baseURL  string            // baseURL 表示 Provider 基础地址
	apiKey   string            // apiKey 表示 Provider 密钥
	modelMap map[string]string // modelMap 表示模型映射关系
	client   *Client           // client 表示底层 OpenAI HTTP 客户端
}

// NewProvider 创建新的 OpenAI Provider
func NewProvider(name string, baseURL string, apiKey string, modelMap map[string]string, timeout time.Duration) *Provider {
	// 补齐默认 Provider 名称
	finalName := name
	if finalName == "" {
		finalName = "openai"
	}

	return &Provider{
		name:     finalName,
		baseURL:  baseURL,
		apiKey:   apiKey,
		modelMap: cloneModelMap(modelMap),
		client:   NewClient(baseURL, apiKey, timeout),
	}
}

// NewProviderWithClient 使用指定客户端创建 OpenAI Provider
func NewProviderWithClient(name string, baseURL string, apiKey string, modelMap map[string]string, client *Client) *Provider {
	// 补齐默认 Provider 名称和客户端
	finalName := name
	if finalName == "" {
		finalName = "openai"
	}
	finalClient := client
	if finalClient == nil {
		finalClient = NewClient(baseURL, apiKey, 30*time.Second)
	}

	return &Provider{
		name:     finalName,
		baseURL:  baseURL,
		apiKey:   apiKey,
		modelMap: cloneModelMap(modelMap),
		client:   finalClient,
	}
}

// Name 返回 Provider 名称
func (p *Provider) Name() string {
	return p.name
}

// Type 返回 Provider 类型
func (p *Provider) Type() string {
	return "openai"
}

// Chat 执行非流式聊天请求
func (p *Provider) Chat(ctx context.Context, req *schema.Request) (*schema.Response, error) {
	// 校验输入请求
	if req == nil {
		return nil, fmt.Errorf("请求不能为空")
	}

	// 转换模型名称与请求结构
	model := p.resolveModel(req.Model)
	openaiReq := ToOpenAIChatCompletionRequest(req)
	openaiReq.Model = model

	// 调用 OpenAI 接口
	openaiResp, err := p.client.ChatCompletion(ctx, openaiReq)
	if err != nil {
		return nil, err
	}

	// 转换为统一响应结构
	resp := ToSchemaResponse(req.RequestID, p.name, openaiResp)
	if resp == nil {
		return nil, fmt.Errorf("openai 响应为空")
	}

	return resp, nil
}

// ChatStream 执行流式聊天请求
func (p *Provider) ChatStream(ctx context.Context, req *schema.Request) (<-chan schema.StreamEvent, <-chan error) {
	// 创建统一流式事件通道和错误通道
	eventCh := make(chan schema.StreamEvent, 8)
	errCh := make(chan error, 1)

	// 校验输入请求
	if req == nil {
		errCh <- fmt.Errorf("请求不能为空")
		close(eventCh)
		close(errCh)
		return eventCh, errCh
	}

	// 转换模型名称与请求结构
	model := p.resolveModel(req.Model)
	openaiReq := ToOpenAIChatCompletionRequest(req)
	openaiReq.Model = model
	openaiReq.Stream = true

	// 调用 OpenAI 流式接口并转换为统一事件
	openaiStream, openaiErrCh := p.client.ChatCompletionStream(ctx, openaiReq)
	go func() {
		defer close(eventCh)
		defer close(errCh)

		for openaiStream != nil || openaiErrCh != nil {
			select {
			case chunk, ok := <-openaiStream:
				if !ok {
					openaiStream = nil
					continue
				}

				// 处理结束信号
				if chunk != nil && chunk.Object == "chat.completion.chunk" && len(chunk.Choices) == 0 && chunk.Usage == nil {
					eventCh <- schema.StreamEvent{
						RequestID:    req.RequestID,
						Provider:     p.name,
						Model:        model,
						Done:         true,
						FinishReason: "stop",
					}
					continue
				}

				event := ToSchemaStreamEvent(req.RequestID, p.name, chunk)
				if event == nil {
					continue
				}
				if event.Model == "" {
					event.Model = model
				}
				event.Provider = p.name
				eventCh <- *event
			case err, ok := <-openaiErrCh:
				if !ok {
					openaiErrCh = nil
					continue
				}
				if err != nil {
					errCh <- err
					return
				}
			}
		}
	}()

	return eventCh, errCh
}

// resolveModel 根据模型映射表解析目标模型
func (p *Provider) resolveModel(model string) string {
	// 优先使用配置中的模型映射
	if mappedModel, ok := p.modelMap[model]; ok && mappedModel != "" {
		return mappedModel
	}

	return model
}

// cloneModelMap 复制模型映射表
func cloneModelMap(modelMap map[string]string) map[string]string {
	// 处理空映射表
	if len(modelMap) == 0 {
		return nil
	}

	result := make(map[string]string, len(modelMap))
	for key, value := range modelMap {
		result[key] = value
	}

	return result
}
