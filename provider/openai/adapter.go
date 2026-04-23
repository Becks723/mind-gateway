package openai

import "github.com/Becks723/mind-gateway/core/schema"

// ToOpenAIChatCompletionRequest 将内部请求转换为 OpenAI 聊天请求
func ToOpenAIChatCompletionRequest(req *schema.Request) *ChatCompletionRequest {
	// 处理空请求
	if req == nil {
		return nil
	}

	// 转换统一消息结构
	messages := make([]ChatMessage, 0, len(req.Messages))
	for _, message := range req.Messages {
		messages = append(messages, ChatMessage{
			Role:    message.Role,
			Content: message.Content,
			Name:    message.Name,
		})
	}

	return &ChatCompletionRequest{
		Model:       req.Model,
		Messages:    messages,
		Stream:      req.Stream,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
		Metadata:    req.Metadata,
	}
}

// ToSchemaResponse 将 OpenAI 聊天响应转换为内部统一响应
func ToSchemaResponse(requestID string, providerName string, resp *ChatCompletionResponse) *schema.Response {
	// 处理空响应
	if resp == nil {
		return nil
	}

	// 提取首个候选结果
	outputText := ""
	finishReason := ""
	messages := make([]schema.Message, 0, 1)
	if len(resp.Choices) > 0 {
		outputText = resp.Choices[0].Message.Content
		finishReason = resp.Choices[0].FinishReason
		messages = append(messages, schema.Message{
			Role:    resp.Choices[0].Message.Role,
			Content: resp.Choices[0].Message.Content,
			Name:    resp.Choices[0].Message.Name,
		})
	}

	return &schema.Response{
		RequestID:    requestID,
		Provider:     providerName,
		Model:        resp.Model,
		OutputText:   outputText,
		Messages:     messages,
		FinishReason: finishReason,
		Usage: schema.Usage{
			InputTokens:  resp.Usage.PromptTokens,
			OutputTokens: resp.Usage.CompletionTokens,
			TotalTokens:  resp.Usage.TotalTokens,
		},
	}
}

// ToSchemaStreamEvent 将 OpenAI 流式响应转换为内部统一流式事件
func ToSchemaStreamEvent(requestID string, providerName string, resp *ChatCompletionChunkResponse) *schema.StreamEvent {
	// 处理空响应
	if resp == nil {
		return nil
	}

	event := &schema.StreamEvent{
		RequestID: requestID,
		Provider:  providerName,
		Model:     resp.Model,
	}

	// 提取首个候选的增量内容
	if len(resp.Choices) > 0 {
		event.Delta = resp.Choices[0].Delta.Content
		event.FinishReason = resp.Choices[0].FinishReason
	}

	// 提取可选使用量
	if resp.Usage != nil {
		event.Usage = &schema.Usage{
			InputTokens:  resp.Usage.PromptTokens,
			OutputTokens: resp.Usage.CompletionTokens,
			TotalTokens:  resp.Usage.TotalTokens,
		}
	}

	return event
}
