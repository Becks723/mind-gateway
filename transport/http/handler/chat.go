package handler

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Becks723/mind-gateway/core/schema"
	"github.com/Becks723/mind-gateway/provider/openai"
	"github.com/valyala/fasthttp"
)

// ChatGateway 定义聊天请求所需的最小网关接口
type ChatGateway interface {
	HandleChat(ctx context.Context, req *schema.Request) (*schema.Response, error)
	HandleChatStream(ctx context.Context, req *schema.Request) (<-chan schema.StreamEvent, <-chan error, error)
}

// ChatCompletion 处理非流式聊天补全请求
func ChatCompletion(gateway ChatGateway) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		// 校验请求方法
		if !ctx.IsPost() {
			ctx.Response.Header.Set("Allow", fasthttp.MethodPost)
			WriteError(ctx, fasthttp.StatusMethodNotAllowed, "方法不被允许")
			return
		}

		// 解析 OpenAI-compatible 请求体
		var reqBody openai.ChatCompletionRequest
		if err := json.Unmarshal(ctx.PostBody(), &reqBody); err != nil {
			WriteError(ctx, fasthttp.StatusBadRequest, "请求体不是合法 JSON")
			return
		}

		// 校验请求必要字段
		if len(reqBody.Messages) == 0 {
			WriteError(ctx, fasthttp.StatusBadRequest, "messages 不能为空")
			return
		}

		// 转换为内部请求对象
		internalReq := &schema.Request{
			RequestID:   requestIDFromContext(ctx),
			Model:       reqBody.Model,
			Messages:    toCoreMessages(reqBody.Messages),
			Stream:      reqBody.Stream,
			Temperature: reqBody.Temperature,
			MaxTokens:   reqBody.MaxTokens,
			Metadata:    reqBody.Metadata,
			VirtualKey:  virtualKeyFromContext(ctx),
			StartedAt:   time.Now(),
		}

		// 调用网关核心执行链路
		if internalReq.Stream {
			handleStreamingChatCompletion(ctx, gateway, internalReq)
			return
		}

		// 调用网关核心执行链路
		resp, err := gateway.HandleChat(ctx, internalReq)
		if err != nil {
			WriteErrorFrom(ctx, err)
			return
		}

		// 转换为 OpenAI-compatible 响应
		responseBody := openai.ChatCompletionResponse{
			ID:      resp.RequestID,
			Object:  "chat.completion",
			Created: time.Now().Unix(),
			Model:   resp.Model,
			Choices: []openai.ChatChoice{
				{
					Index: 0,
					Message: openai.ChatMessage{
						Role:    "assistant",
						Content: resp.OutputText,
					},
					FinishReason: resp.FinishReason,
				},
			},
			Usage: openai.Usage{
				PromptTokens:     resp.Usage.InputTokens,
				CompletionTokens: resp.Usage.OutputTokens,
				TotalTokens:      resp.Usage.TotalTokens,
			},
		}

		// 输出 JSON 响应
		body, err := json.Marshal(responseBody)
		if err != nil {
			WriteError(ctx, fasthttp.StatusInternalServerError, "编码响应失败")
			return
		}

		ctx.Response.Header.SetContentType("application/json")
		ctx.SetStatusCode(fasthttp.StatusOK)
		ctx.SetBody(body)
	}
}

// handleStreamingChatCompletion 处理 SSE 流式聊天补全请求
func handleStreamingChatCompletion(ctx *fasthttp.RequestCtx, gateway ChatGateway, req *schema.Request) {
	// 建立流式请求
	eventCh, errCh, err := gateway.HandleChatStream(ctx, req)
	if err != nil {
		WriteErrorFrom(ctx, err)
		return
	}

	// 设置 SSE 响应头
	ctx.Response.Header.SetContentType("text/event-stream")
	ctx.Response.Header.Set("Cache-Control", "no-cache")
	ctx.Response.Header.Set("Connection", "keep-alive")
	ctx.SetStatusCode(fasthttp.StatusOK)

	// 按 SSE 格式持续输出事件
	ctx.SetBodyStreamWriter(func(writer *bufio.Writer) {
		for eventCh != nil || errCh != nil {
			select {
			case event, ok := <-eventCh:
				if !ok {
					eventCh = nil
					continue
				}

				if event.Done {
					_, _ = writer.WriteString("data: [DONE]\n\n")
					_ = writer.Flush()
					continue
				}

				payload, buildErr := buildStreamingChunk(event)
				if buildErr != nil {
					writeStreamError(writer, buildErr)
					return
				}
				_, _ = writer.WriteString("data: " + payload + "\n\n")
				_ = writer.Flush()
			case streamErr, ok := <-errCh:
				if !ok {
					errCh = nil
					continue
				}
				if streamErr != nil {
					writeStreamError(writer, streamErr)
					return
				}
			}
		}
	})
}

// buildStreamingChunk 构造 OpenAI-compatible 的流式事件 JSON
func buildStreamingChunk(event schema.StreamEvent) (string, error) {
	// 构造兼容 OpenAI 的 chunk 响应
	body, err := json.Marshal(openai.ChatCompletionChunkResponse{
		ID:      event.RequestID,
		Object:  "chat.completion.chunk",
		Created: time.Now().Unix(),
		Model:   event.Model,
		Choices: []openai.ChatChunkChoice{
			{
				Index: 0,
				Delta: openai.ChatMessageDelta{
					Role:    "assistant",
					Content: event.Delta,
				},
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("编码流式响应失败: %w", err)
	}

	return string(body), nil
}

// writeStreamError 向 SSE 流中输出错误事件
func writeStreamError(writer *bufio.Writer, err error) {
	// 构造统一的错误输出事件
	body, marshalErr := json.Marshal(APIError{
		Error: ErrorBody{
			Message: err.Error(),
			Type:    schema.ErrorTypeInternal,
		},
	})
	if marshalErr != nil {
		_, _ = writer.WriteString("data: {\"error\":{\"message\":\"流式请求失败\",\"type\":\"internal_error\"}}\n\n")
		_, _ = writer.WriteString("data: [DONE]\n\n")
		_ = writer.Flush()
		return
	}

	_, _ = writer.WriteString("data: " + string(body) + "\n\n")
	_, _ = writer.WriteString("data: [DONE]\n\n")
	_ = writer.Flush()
}

// toCoreMessages 将外部消息结构转换为内部消息结构
func toCoreMessages(messages []openai.ChatMessage) []schema.Message {
	// 逐条转换消息结构
	result := make([]schema.Message, 0, len(messages))
	for _, message := range messages {
		result = append(result, schema.Message{
			Role:    message.Role,
			Content: message.Content,
			Name:    message.Name,
		})
	}

	return result
}

// requestIDFromContext 从请求上下文中读取请求 ID
func requestIDFromContext(ctx *fasthttp.RequestCtx) string {
	// 优先从中间件注入的上下文读取请求 ID
	requestID, _ := ctx.UserValue("request_id").(string)
	return requestID
}

// virtualKeyFromContext 从请求上下文中读取虚拟密钥
func virtualKeyFromContext(ctx *fasthttp.RequestCtx) string {
	// 优先读取自定义头中的虚拟密钥
	virtualKey := strings.TrimSpace(string(ctx.Request.Header.Peek("X-Mind-Virtual-Key")))
	if virtualKey != "" {
		return virtualKey
	}

	// 兼容从 Authorization 的 Bearer 中读取虚拟密钥
	authorization := strings.TrimSpace(string(ctx.Request.Header.Peek("Authorization")))
	if authorization == "" {
		return ""
	}
	if !strings.HasPrefix(strings.ToLower(authorization), "bearer ") {
		return ""
	}

	return strings.TrimSpace(authorization[len("Bearer "):])
}
