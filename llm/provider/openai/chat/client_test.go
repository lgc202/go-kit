package chat

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/lgc202/go-kit/llm"
	"github.com/lgc202/go-kit/llm/schema"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// TestChat_BasicRequest 测试基本的 chat 请求
func TestChat_BasicRequest(t *testing.T) {
	t.Parallel()

	httpClient := &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			body := `{
  "id":"chatcmpl-123",
  "object":"chat.completion",
  "created": 1677652288,
  "model":"gpt-4o-mini",
  "choices":[{
    "index":0,
    "message":{"role":"assistant","content":"Hello!"},
    "finish_reason":"stop"
  }],
  "usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}
}`

			h := make(http.Header)
			h.Set("Content-Type", "application/json")
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(body)),
				Header:     h,
				Request:    r,
			}, nil
		}),
	}

	c, err := New(Config{
		BaseURL:        "https://api.openai.com/v1",
		APIKey:         "test-key",
		HTTPClient:     httpClient,
		DefaultOptions: []llm.ChatOption{llm.WithModel("gpt-4o-mini")},
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	resp, err := c.Chat(context.Background(), []schema.Message{
		schema.UserMessage("Hello"),
	})
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	if resp.ID != "chatcmpl-123" {
		t.Errorf("ID = %q, want %q", resp.ID, "chatcmpl-123")
	}
	if resp.Model != "gpt-4o-mini" {
		t.Errorf("Model = %q, want %q", resp.Model, "gpt-4o-mini")
	}
	if len(resp.Choices) != 1 {
		t.Fatalf("len(Choices) = %d, want 1", len(resp.Choices))
	}
	if resp.Choices[0].Message.Text() != "Hello!" {
		t.Errorf("Message.Text() = %q, want %q", resp.Choices[0].Message.Text(), "Hello!")
	}
	if resp.Choices[0].FinishReason != schema.FinishReasonStop {
		t.Errorf("FinishReason = %v, want %v", resp.Choices[0].FinishReason, schema.FinishReasonStop)
	}
	if resp.Usage.TotalTokens != 15 {
		t.Errorf("TotalTokens = %d, want 15", resp.Usage.TotalTokens)
	}
}

// TestChat_MultimodalAndToolsRequest 测试多模态和工具调用
func TestChat_MultimodalAndToolsRequest(t *testing.T) {
	t.Parallel()

	var gotReq map[string]any

	httpClient := &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			if err := json.NewDecoder(r.Body).Decode(&gotReq); err != nil {
				return &http.Response{
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(strings.NewReader(err.Error())),
					Header:     make(http.Header),
					Request:    r,
				}, nil
			}

			body := `{
  "id":"abc",
  "created": 1,
  "model":"gpt-4o-mini",
  "choices":[{"index":0,"finish_reason":"stop","message":{"role":"assistant","content":"ok"}}],
  "usage":{"prompt_tokens":1,"completion_tokens":2,"total_tokens":3}
}`

			h := make(http.Header)
			h.Set("Content-Type", "application/json")
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(body)),
				Header:     h,
				Request:    r,
			}, nil
		}),
	}

	c, err := New(Config{
		BaseURL:        "https://example.test/v1",
		APIKey:         "tok",
		HTTPClient:     httpClient,
		DefaultOptions: []llm.ChatOption{llm.WithModel("gpt-4o-mini")},
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	tools := []schema.Tool{
		{
			Type: schema.ToolTypeFunction,
			Function: schema.FunctionDefinition{
				Name:       "get_weather",
				Parameters: json.RawMessage(`{"type":"object","properties":{"city":{"type":"string"}}}`),
			},
		},
	}

	_, err = c.Chat(context.Background(), []schema.Message{
		{
			Role: schema.RoleUser,
			Content: []schema.ContentPart{
				schema.TextPart("What's in this image?"),
				schema.ImageURLPart("https://example.com/cat.png"),
			},
		},
	}, llm.WithTools(tools...), llm.WithToolChoice(schema.ToolChoice{Mode: schema.ToolChoiceAuto}))
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	// 验证请求格式
	msgs, _ := gotReq["messages"].([]any)
	if len(msgs) != 1 {
		t.Fatalf("len(messages) = %d, want 1", len(msgs))
	}
	m0, _ := msgs[0].(map[string]any)
	content, _ := m0["content"].([]any)
	if len(content) != 2 {
		t.Fatalf("len(content) = %d, want 2 (text + image)", len(content))
	}

	if gotReq["tool_choice"] != "auto" {
		t.Errorf("tool_choice = %v, want %v", gotReq["tool_choice"], "auto")
	}
	if _, ok := gotReq["tools"]; !ok {
		t.Error("tools field missing")
	}
}

// TestChatStream_Basic 测试基本的流式响应
func TestChatStream_Basic(t *testing.T) {
	t.Parallel()

	httpClient := &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			body := `data: {"choices":[{"index":0,"delta":{"content":"Hello"}}]}

data: {"choices":[{"index":0,"delta":{"content":" World"}}]}

data: {"choices":[{"index":0,"delta":{}}],"finish_reason":"stop"}

data: [DONE]

`

			h := make(http.Header)
			h.Set("Content-Type", "text/event-stream")
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(body)),
				Header:     h,
				Request:    r,
			}, nil
		}),
	}

	c, err := New(Config{
		BaseURL:        "https://api.openai.com/v1",
		APIKey:         "test-key",
		HTTPClient:     httpClient,
		DefaultOptions: []llm.ChatOption{llm.WithModel("gpt-4o-mini")},
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	stream, err := c.ChatStream(context.Background(), []schema.Message{
		schema.UserMessage("Say hello"),
	})
	if err != nil {
		t.Fatalf("ChatStream() error = %v", err)
	}
	defer stream.Close()

	var sb strings.Builder
	for {
		event, err := stream.Recv()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			t.Fatalf("Recv() error = %v", err)
		}

		if event.Type == schema.StreamEventDelta {
			sb.WriteString(event.Delta)
		}
	}

	if got := sb.String(); got != "Hello World" {
		t.Errorf("Received text = %q, want %q", got, "Hello World")
	}
}

// TestChat_APIErrorResponse 测试 API 错误响应
func TestChat_APIErrorResponse(t *testing.T) {
	t.Parallel()

	httpClient := &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			body := `{
  "error":{
    "message":"Invalid API key",
    "type":"invalid_request_error",
    "code":"invalid_api_key"
  }
}`

			h := make(http.Header)
			h.Set("Content-Type", "application/json")
			return &http.Response{
				StatusCode: http.StatusUnauthorized,
				Body:       io.NopCloser(strings.NewReader(body)),
				Header:     h,
				Request:    r,
			}, nil
		}),
	}

	c, err := New(Config{
		APIKey:         "invalid-key",
		HTTPClient:     httpClient,
		DefaultOptions: []llm.ChatOption{llm.WithModel("gpt-4o-mini")},
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	_, err = c.Chat(context.Background(), []schema.Message{
		schema.UserMessage("test"),
	})
	if err == nil {
		t.Fatal("Chat() should return error for invalid API key")
	}

	apiErr, ok := llm.AsAPIError(err)
	if !ok {
		t.Fatalf("Error should be APIError, got %T", err)
	}

	if apiErr.StatusCode != http.StatusUnauthorized {
		t.Errorf("StatusCode = %d, want %d", apiErr.StatusCode, http.StatusUnauthorized)
	}
	if apiErr.Code != "invalid_api_key" {
		t.Errorf("Code = %q, want %q", apiErr.Code, "invalid_api_key")
	}
	if !llm.IsAuth(err) {
		t.Error("IsAuth() should return true")
	}
}

// TestChat_ExtraFieldsOverride 测试扩展字段覆盖保护
func TestChat_ExtraFieldsOverride(t *testing.T) {
	t.Parallel()

	var gotReq map[string]any
	httpClient := &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			if err := json.NewDecoder(r.Body).Decode(&gotReq); err != nil {
				return &http.Response{
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(strings.NewReader(err.Error())),
					Header:     make(http.Header),
					Request:    r,
				}, nil
			}

			body := `{
  "id":"abc",
  "created": 1,
  "model":"gpt-4o-mini",
  "choices":[{"index":0,"finish_reason":"stop","message":{"role":"assistant","content":"ok"}}],
  "usage":{"prompt_tokens":1,"completion_tokens":2,"total_tokens":3}
}`

			h := make(http.Header)
			h.Set("Content-Type", "application/json")
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(body)),
				Header:     h,
				Request:    r,
			}, nil
		}),
	}

	c, err := New(Config{
		BaseURL:        "https://example.test/v1",
		APIKey:         "tok",
		HTTPClient:     httpClient,
		DefaultOptions: []llm.ChatOption{llm.WithModel("gpt-4o-mini")},
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// 测试不允许覆盖时应该报错
	_, err = c.Chat(context.Background(), []schema.Message{
		schema.UserMessage("Hi"),
	}, llm.WithExtraField("model", "override-model"))
	if err == nil || !strings.Contains(err.Error(), "conflicts") {
		t.Fatalf("Expected conflict error, got: %v", err)
	}

	// 测试允许覆盖时应该成功
	_, err = c.Chat(context.Background(), []schema.Message{
		schema.UserMessage("Hi"),
	}, llm.WithAllowExtraFieldOverride(true), llm.WithExtraField("model", "override-model"))
	if err != nil {
		t.Fatalf("Chat() with override error = %v", err)
	}
	if gotReq["model"] != "override-model" {
		t.Errorf("model = %v, want %v", gotReq["model"], "override-model")
	}
}

// TestChat_ResponseHook 测试响应钩子
func TestChat_ResponseHook(t *testing.T) {
	t.Parallel()

	httpClient := &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			body := `{
  "id":"abc",
  "created": 1,
  "model":"gpt-4o-mini",
  "choices":[{"index":0,"finish_reason":"stop","message":{"role":"assistant","content":"ok"}}],
  "usage":{"prompt_tokens":1,"completion_tokens":2,"total_tokens":3},
  "x_custom_field":"custom_value"
}`

			h := make(http.Header)
			h.Set("Content-Type", "application/json")
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(body)),
				Header:     h,
				Request:    r,
			}, nil
		}),
	}

	c, err := New(Config{
		BaseURL:        "https://example.test/v1",
		APIKey:         "tok",
		HTTPClient:     httpClient,
		DefaultOptions: []llm.ChatOption{llm.WithModel("gpt-4o-mini")},
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	resp, err := c.Chat(context.Background(), []schema.Message{schema.UserMessage("Hi")},
		llm.WithResponseHook(func(dst *schema.ChatResponse, raw json.RawMessage) error {
			var m map[string]any
			if err := json.Unmarshal(raw, &m); err != nil {
				return err
			}
			if dst.ExtraFields == nil {
				dst.ExtraFields = make(map[string]any)
			}
			dst.ExtraFields["x_custom_field"] = m["x_custom_field"]
			return nil
		}),
	)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	if resp.ExtraFields == nil || resp.ExtraFields["x_custom_field"] != "custom_value" {
		t.Errorf("ExtraFields not populated, got: %#v", resp.ExtraFields)
	}
	if len(resp.Raw) != 0 {
		t.Errorf("Raw should be empty when KeepRaw=false, got: %s", string(resp.Raw))
	}
}

// TestChat_ErrorHook 测试错误钩子
func TestChat_ErrorHook(t *testing.T) {
	t.Parallel()

	httpClient := &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			body := `{"error":{"message":"rate limit","code":"rate_limit_error"}}`
			h := make(http.Header)
			h.Set("Content-Type", "application/json")
			return &http.Response{
				StatusCode: http.StatusTooManyRequests,
				Body:       io.NopCloser(strings.NewReader(body)),
				Header:     h,
				Request:    r,
			}, nil
		}),
	}

	c, err := New(Config{
		BaseURL:        "https://example.test/v1",
		APIKey:         "tok",
		HTTPClient:     httpClient,
		DefaultOptions: []llm.ChatOption{llm.WithModel("gpt-4o-mini")},
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	want := errors.New("custom rate limit error")
	_, err = c.Chat(context.Background(), []schema.Message{schema.UserMessage("Hi")},
		llm.WithErrorHook(func(provider llm.Provider, statusCode int, body []byte) error {
			if provider != llm.ProviderOpenAI || statusCode != http.StatusTooManyRequests || len(body) == 0 {
				t.Errorf("Unexpected hook args: provider=%q status=%d body=%q", provider, statusCode, string(body))
			}
			return want
		}),
	)
	if !errors.Is(err, want) {
		t.Fatalf("Error = %v, want %v", err, want)
	}
}
