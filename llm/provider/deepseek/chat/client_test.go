package chat

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/lgc202/go-kit/llm"
	"github.com/lgc202/go-kit/llm/schema"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// TestChat_ReasoningContent 测试 DeepSeek 特有的 reasoning_content 字段
func TestChat_ReasoningContent(t *testing.T) {
	t.Parallel()

	httpClient := &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			body := `{
  "id":"abc",
  "created": 1,
  "model":"deepseek-chat",
  "choices":[{
    "index":0,
    "finish_reason":"stop",
    "message":{
      "role":"assistant",
      "content":"Final answer: The capital of France is Paris",
      "reasoning_content":"Let me think... France is a country in Europe. The capital city is Paris."
    }
  }],
  "usage":{"prompt_tokens":10,"completion_tokens":20,"total_tokens":30}
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
		APIKey:         "tok",
		HTTPClient:     httpClient,
		DefaultOptions: []llm.ChatOption{llm.WithModel("deepseek-chat")},
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	resp, err := c.Chat(context.Background(), []schema.Message{
		schema.UserMessage("What is the capital of France?"),
	})
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	// 验证 reasoning_content 字段被正确解析
	gotReasoning := resp.Choices[0].Message.ReasoningContent
	wantReasoning := "Let me think... France is a country in Europe. The capital city is Paris."
	if gotReasoning != wantReasoning {
		t.Errorf("ReasoningContent = %q, want %q", gotReasoning, wantReasoning)
	}

	// 验证普通 content 字段也被正确解析
	gotText := resp.Choices[0].Message.Text()
	wantText := "Final answer: The capital of France is Paris"
	if gotText != wantText {
		t.Errorf("Text() = %q, want %q", gotText, wantText)
	}

	// 验证 usage 统计
	if resp.Usage.TotalTokens != 30 {
		t.Errorf("TotalTokens = %d, want 30", resp.Usage.TotalTokens)
	}
}

// TestChatStream_ReasoningContent 测试流式响应中的 reasoning_content
func TestChatStream_ReasoningContent(t *testing.T) {
	t.Parallel()

	httpClient := &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			h := make(http.Header)
			h.Set("Content-Type", "text/event-stream")

			// DeepSeek 流式响应：先输出 reasoning_content，再输出最终答案
			body := "data: {\"choices\":[{\"index\":0,\"delta\":{\"reasoning_content\":\"Thinking...\"}}]}\n\n" +
				"data: {\"choices\":[{\"index\":0,\"delta\":{\"content\":\"Answer\"}}]}\n\n" +
				"data: {\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n" +
				"data: [DONE]\n\n"

			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(body)),
				Header:     h,
				Request:    r,
			}, nil
		}),
	}

	c, err := New(Config{
		APIKey:         "tok",
		HTTPClient:     httpClient,
		DefaultOptions: []llm.ChatOption{llm.WithModel("deepseek-chat")},
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	st, err := c.ChatStream(context.Background(), []schema.Message{
		schema.UserMessage("What is 2+2?"),
	})
	if err != nil {
		t.Fatalf("ChatStream() error = %v", err)
	}
	// 确保流被关闭
	t.Cleanup(func() { st.Close() })

	// 第一个事件：reasoning_content
	ev, err := st.Recv()
	if err != nil {
		t.Fatalf("Recv() error = %v", err)
	}
	if ev.Type != schema.StreamEventDelta {
		t.Errorf("First event Type = %v, want %v", ev.Type, schema.StreamEventDelta)
	}
	if ev.Reasoning != "Thinking..." {
		t.Errorf("First event Reasoning = %q, want %q", ev.Reasoning, "Thinking...")
	}

	// 第二个事件：content
	ev, err = st.Recv()
	if err != nil {
		t.Fatalf("Recv() error = %v", err)
	}
	if ev.Delta != "Answer" {
		t.Errorf("Second event Delta = %q, want %q", ev.Delta, "Answer")
	}

	// 第三个事件：完成
	ev, err = st.Recv()
	if err != nil && err.Error() != "EOF" {
		t.Fatalf("Recv() error = %v", err)
	}
	if ev.FinishReason == nil || *ev.FinishReason != schema.FinishReasonStop {
		t.Errorf("FinishReason = %v, want %v", ev.FinishReason, schema.FinishReasonStop)
	}
}

// TestChat_ReasoningContentEmpty 测试 reasoning_content 可以为空
func TestChat_ReasoningContentEmpty(t *testing.T) {
	t.Parallel()

	httpClient := &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			body := `{
  "id":"abc",
  "created": 1,
  "model":"deepseek-chat",
  "choices":[{
    "index":0,
    "finish_reason":"stop",
    "message":{"role":"assistant","content":"Simple answer"}
  }],
  "usage":{"prompt_tokens":5,"completion_tokens":5,"total_tokens":10}
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
		APIKey:         "tok",
		HTTPClient:     httpClient,
		DefaultOptions: []llm.ChatOption{llm.WithModel("deepseek-chat")},
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	resp, err := c.Chat(context.Background(), []schema.Message{
		schema.UserMessage("Hi"),
	})
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	// reasoning_content 为空时不应该报错
	if resp.Choices[0].Message.ReasoningContent != "" {
		t.Errorf("ReasoningContent = %q, want empty", resp.Choices[0].Message.ReasoningContent)
	}
	if resp.Choices[0].Message.Text() != "Simple answer" {
		t.Errorf("Text() = %q, want %q", resp.Choices[0].Message.Text(), "Simple answer")
	}
}
