package handler

import (
	"context"
	"encoding/json"
	"time"

	"github.com/Becks723/mind-gateway/core/schema"
	"github.com/Becks723/mind-gateway/provider/openai"
	"github.com/valyala/fasthttp"
)

// ChatGateway 定义聊天请求所需的最小网关接口
type ChatGateway interface {
	HandleChat(ctx context.Context, req *schema.Request) (*schema.Response, error)
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
			StartedAt:   time.Now(),
		}

		// 调用网关核心执行链路
		resp, err := gateway.HandleChat(ctx, internalReq)
		if err != nil {
			WriteError(ctx, fasthttp.StatusInternalServerError, err.Error())
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
