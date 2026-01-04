package chat

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/lgc202/go-kit/llm"
	"github.com/lgc202/go-kit/llm/schema"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestChat_RequestAndResponseMapping(t *testing.T) {
	t.Parallel()

	var gotPath string
	var gotAuth string
	var gotReq map[string]any

	httpClient := &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			gotPath = r.URL.Path
			gotAuth = r.Header.Get("Authorization")

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
  "model":"deepseek-chat",
  "choices":[{"index":0,"finish_reason":"stop","message":{"role":"assistant","content":"ok","reasoning_content":"r","tool_calls":[{"id":"tc1","type":"function","function":{"name":"f","arguments":"{}"}}]}}],
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
		BaseURL:        "https://example.test",
		APIKey:         "tok",
		HTTPClient:     httpClient,
		DefaultOptions: []llm.ChatOption{llm.WithModel("deepseek-chat")},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	resp, err := c.Chat(
		context.Background(),
		[]schema.Message{
			schema.SystemMessage("You are a helpful assistant"),
			schema.UserMessage("Hi"),
		},
		WithThinking(false),
		llm.WithResponseFormat(schema.ResponseFormat{Type: "text"}),
		llm.WithFrequencyPenalty(0),
		llm.WithPresencePenalty(0),
		llm.WithMaxTokens(4096),
		llm.WithTemperature(1),
		llm.WithTopP(1),
	)
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}

	if gotPath != "/chat/completions" {
		t.Fatalf("path: got %q", gotPath)
	}
	if gotAuth != "Bearer tok" {
		t.Fatalf("auth: got %q", gotAuth)
	}

	if gotReq["model"] != "deepseek-chat" {
		t.Fatalf("model: got %#v", gotReq["model"])
	}
	if gotReq["stream"] != false {
		t.Fatalf("stream: got %#v", gotReq["stream"])
	}
	if gotReq["max_tokens"] != float64(4096) {
		t.Fatalf("max_tokens: got %#v", gotReq["max_tokens"])
	}
	if gotReq["temperature"] != float64(1) {
		t.Fatalf("temperature: got %#v", gotReq["temperature"])
	}
	if gotReq["top_p"] != float64(1) {
		t.Fatalf("top_p: got %#v", gotReq["top_p"])
	}

	thinking, _ := gotReq["thinking"].(map[string]any)
	if thinking["type"] != "disabled" {
		t.Fatalf("thinking.type: got %#v", gotReq["thinking"])
	}

	rf, _ := gotReq["response_format"].(map[string]any)
	if rf["type"] != "text" {
		t.Fatalf("response_format.type: got %#v", gotReq["response_format"])
	}

	if len(resp.Choices) != 1 {
		t.Fatalf("choices: got %d", len(resp.Choices))
	}
	if resp.Choices[0].Message.Text() != "ok" {
		t.Fatalf("content: got %q", resp.Choices[0].Message.Text())
	}
	if resp.Choices[0].Message.ReasoningContent != "r" {
		t.Fatalf("reasoning_content: got %q", resp.Choices[0].Message.ReasoningContent)
	}
	if len(resp.Choices[0].Message.ToolCalls) != 1 {
		t.Fatalf("tool_calls: got %d", len(resp.Choices[0].Message.ToolCalls))
	}
	if resp.Choices[0].Message.ToolCalls[0].Function.Name != "f" {
		t.Fatalf("tool_call.function.name: got %q", resp.Choices[0].Message.ToolCalls[0].Function.Name)
	}
}

func TestChatStream_BasicDelta(t *testing.T) {
	t.Parallel()

	httpClient := &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			h := make(http.Header)
			h.Set("Content-Type", "text/event-stream")

			body := "data: {\"choices\":[{\"index\":0,\"delta\":{\"content\":\"Hi\",\"reasoning_content\":\"R\"}}]}\n\n" +
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
		BaseURL:        "https://example.test",
		APIKey:         "tok",
		HTTPClient:     httpClient,
		DefaultOptions: []llm.ChatOption{llm.WithModel("deepseek-chat")},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	st, err := c.ChatStream(context.Background(), []schema.Message{schema.UserMessage("Hi")})
	if err != nil {
		t.Fatalf("ChatStream: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	ev, err := st.Recv()
	if err != nil {
		t.Fatalf("Recv: %v", err)
	}
	if ev.Type != schema.StreamEventDelta || ev.Delta != "Hi" || ev.Reasoning != "R" {
		t.Fatalf("event: %#v", ev)
	}
}

func TestChatStream_EventHookCanEnrichExtraFields(t *testing.T) {
	t.Parallel()

	httpClient := &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			h := make(http.Header)
			h.Set("Content-Type", "text/event-stream")

			body := "data: {\"choices\":[{\"index\":0,\"delta\":{\"content\":\"Hi\"}}],\"x_provider_meta\":{\"foo\":\"bar\"}}\n\n" +
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
		BaseURL:        "https://example.test",
		APIKey:         "tok",
		HTTPClient:     httpClient,
		DefaultOptions: []llm.ChatOption{llm.WithModel("deepseek-chat")},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	st, err := c.ChatStream(context.Background(), []schema.Message{schema.UserMessage("Hi")},
		llm.WithStreamEventHook(func(dst *schema.StreamEvent, raw json.RawMessage) error {
			var m map[string]any
			if err := json.Unmarshal(raw, &m); err != nil {
				return err
			}
			if dst.ExtraFields == nil {
				dst.ExtraFields = make(map[string]any)
			}
			dst.ExtraFields["x_provider_meta"] = m["x_provider_meta"]
			return nil
		}),
	)
	if err != nil {
		t.Fatalf("ChatStream: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	ev, err := st.Recv()
	if err != nil {
		t.Fatalf("Recv: %v", err)
	}
	if ev.Type != schema.StreamEventDelta || ev.Delta != "Hi" {
		t.Fatalf("event: %#v", ev)
	}
	if len(ev.Raw) != 0 {
		t.Fatalf("expected Raw to be empty when KeepRaw=false, got: %s", string(ev.Raw))
	}
	if ev.ExtraFields == nil || ev.ExtraFields["x_provider_meta"] == nil {
		t.Fatalf("expected ExtraFields to be enriched, got: %#v", ev.ExtraFields)
	}
}
