package openai

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/valyala/fasthttp"
)

const defaultBaseURL = "https://api.openai.com"

// Client 表示 OpenAI Provider 的底层 HTTP 客户端
type Client struct {
	baseURL      string           // baseURL 表示 OpenAI 接口基础地址
	apiKey       string           // apiKey 表示访问 OpenAI 的密钥
	client       *fasthttp.Client // client 表示非流式 fasthttp 客户端
	streamClient *http.Client     // streamClient 表示流式 HTTP 客户端
}

// NewClient 创建新的 OpenAI HTTP 客户端
func NewClient(baseURL string, apiKey string, timeout time.Duration) *Client {
	// 补齐默认地址和超时配置
	finalBaseURL := strings.TrimRight(baseURL, "/")
	if finalBaseURL == "" {
		finalBaseURL = defaultBaseURL
	}
	finalTimeout := timeout
	if finalTimeout <= 0 {
		finalTimeout = 30 * time.Second
	}

	return &Client{
		baseURL: finalBaseURL,
		apiKey:  apiKey,
		client: &fasthttp.Client{
			ReadTimeout:  finalTimeout,
			WriteTimeout: finalTimeout,
		},
		streamClient: &http.Client{
			Timeout: finalTimeout,
		},
	}
}

// NewClientWithHTTPClient 使用指定的 fasthttp 客户端创建 OpenAI 客户端
func NewClientWithHTTPClient(baseURL string, apiKey string, httpClient *fasthttp.Client) *Client {
	// 补齐默认地址和客户端配置
	finalBaseURL := strings.TrimRight(baseURL, "/")
	if finalBaseURL == "" {
		finalBaseURL = defaultBaseURL
	}
	finalHTTPClient := httpClient
	if finalHTTPClient == nil {
		finalHTTPClient = &fasthttp.Client{}
	}

	return &Client{
		baseURL: finalBaseURL,
		apiKey:  apiKey,
		client:  finalHTTPClient,
		streamClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// NewClientWithStreamClient 使用指定流式 HTTP 客户端创建 OpenAI 客户端
func NewClientWithStreamClient(baseURL string, apiKey string, httpClient *http.Client) *Client {
	// 补齐默认地址和客户端配置
	finalBaseURL := strings.TrimRight(baseURL, "/")
	if finalBaseURL == "" {
		finalBaseURL = defaultBaseURL
	}
	finalHTTPClient := httpClient
	if finalHTTPClient == nil {
		finalHTTPClient = &http.Client{
			Timeout: 30 * time.Second,
		}
	}

	return &Client{
		baseURL:      finalBaseURL,
		apiKey:       apiKey,
		client:       &fasthttp.Client{},
		streamClient: finalHTTPClient,
	}
}

// ChatCompletion 调用 OpenAI 聊天补全接口
func (c *Client) ChatCompletion(ctx context.Context, req *ChatCompletionRequest) (*ChatCompletionResponse, error) {
	// 校验输入请求
	if req == nil {
		return nil, fmt.Errorf("openai 请求不能为空")
	}

	// 编码请求体
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("编码 openai 请求失败: %w", err)
	}

	// 构建 HTTP 请求与响应对象
	httpReq := fasthttp.AcquireRequest()
	httpResp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(httpReq)
	defer fasthttp.ReleaseResponse(httpResp)

	httpReq.SetRequestURI(c.baseURL + "/v1/chat/completions")
	httpReq.Header.SetMethod(fasthttp.MethodPost)
	httpReq.Header.SetContentType("application/json")
	if c.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	httpReq.SetBody(body)

	// 发起带上下文的请求
	if err := c.client.DoTimeout(httpReq, httpResp, requestTimeoutFromContext(ctx)); err != nil {
		return nil, fmt.Errorf("调用 openai 接口失败: %w", err)
	}

	// 处理非成功状态码
	if httpResp.StatusCode() < fasthttp.StatusOK || httpResp.StatusCode() >= fasthttp.StatusMultipleChoices {
		return nil, fmt.Errorf("openai 返回错误状态码: %d", httpResp.StatusCode())
	}

	// 解析响应体
	var response ChatCompletionResponse
	if err := json.Unmarshal(httpResp.Body(), &response); err != nil {
		return nil, fmt.Errorf("解析 openai 响应失败: %w", err)
	}

	return &response, nil
}

// ChatCompletionStream 调用 OpenAI 流式聊天补全接口
func (c *Client) ChatCompletionStream(ctx context.Context, req *ChatCompletionRequest) (<-chan *ChatCompletionChunkResponse, <-chan error) {
	// 创建流式响应通道和错误通道
	streamCh := make(chan *ChatCompletionChunkResponse, 8)
	errCh := make(chan error, 1)

	// 校验输入请求
	if req == nil {
		errCh <- fmt.Errorf("openai 流式请求不能为空")
		close(streamCh)
		close(errCh)
		return streamCh, errCh
	}

	go func() {
		defer close(streamCh)
		defer close(errCh)

		// 编码请求体
		streamReq := *req
		streamReq.Stream = true
		body, err := json.Marshal(&streamReq)
		if err != nil {
			errCh <- fmt.Errorf("编码 openai 流式请求失败: %w", err)
			return
		}

		// 构建 HTTP 请求
		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/chat/completions", strings.NewReader(string(body)))
		if err != nil {
			errCh <- fmt.Errorf("创建 openai 流式请求失败: %w", err)
			return
		}
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Accept", "text/event-stream")
		if c.apiKey != "" {
			httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
		}

		// 发起流式请求
		httpResp, err := c.streamClient.Do(httpReq)
		if err != nil {
			errCh <- fmt.Errorf("调用 openai 流式接口失败: %w", err)
			return
		}
		defer httpResp.Body.Close()

		// 处理非成功状态码
		if httpResp.StatusCode < http.StatusOK || httpResp.StatusCode >= http.StatusMultipleChoices {
			errCh <- fmt.Errorf("openai 流式接口返回错误状态码: %d", httpResp.StatusCode)
			return
		}

		// 逐行解析 SSE 数据
		reader := bufio.NewScanner(httpResp.Body)
		for reader.Scan() {
			line := strings.TrimSpace(reader.Text())
			if line == "" || !strings.HasPrefix(line, "data: ") {
				continue
			}

			payload := strings.TrimSpace(strings.TrimPrefix(line, "data: "))
			if payload == "[DONE]" {
				streamCh <- &ChatCompletionChunkResponse{
					Object: "chat.completion.chunk",
				}
				return
			}

			var chunk ChatCompletionChunkResponse
			if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
				errCh <- fmt.Errorf("解析 openai 流式响应失败: %w", err)
				return
			}
			streamCh <- &chunk
		}

		if err := reader.Err(); err != nil {
			errCh <- fmt.Errorf("读取 openai 流式响应失败: %w", err)
		}
	}()

	return streamCh, errCh
}

// requestTimeoutFromContext 从上下文中解析请求超时
func requestTimeoutFromContext(ctx context.Context) time.Duration {
	// 优先使用上下文截止时间推导超时
	if deadline, ok := ctx.Deadline(); ok {
		timeout := time.Until(deadline)
		if timeout > 0 {
			return timeout
		}
	}

	return 30 * time.Second
}
