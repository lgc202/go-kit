package openai

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

func TestChat_MultimodalAndToolsRequest(t *testing.T) {
	t.Parallel()

	var gotPath string
	var gotReq map[string]any

	httpClient := &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			gotPath = r.URL.Path
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
		BaseURL:    "https://example.test/v1",
		APIKey:     "tok",
		HTTPClient: httpClient,
		DefaultRequest: llm.RequestConfig{
			Model: "gpt-4o-mini",
		},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
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
		t.Fatalf("Chat: %v", err)
	}

	if gotPath != "/v1/chat/completions" {
		t.Fatalf("path: got %q", gotPath)
	}

	msgs, _ := gotReq["messages"].([]any)
	if len(msgs) != 1 {
		t.Fatalf("messages: %#v", gotReq["messages"])
	}
	m0, _ := msgs[0].(map[string]any)
	content, _ := m0["content"].([]any)
	if len(content) != 2 {
		t.Fatalf("content: %#v", m0["content"])
	}

	if gotReq["tool_choice"] != "auto" {
		t.Fatalf("tool_choice: %#v", gotReq["tool_choice"])
	}
	if _, ok := gotReq["tools"]; !ok {
		t.Fatalf("tools missing")
	}
}

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
		BaseURL:    "https://example.test/v1",
		APIKey:     "tok",
		HTTPClient: httpClient,
		DefaultRequest: llm.RequestConfig{
			Model: "gpt-4o-mini",
		},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, err = c.Chat(context.Background(), []schema.Message{
		schema.UserMessage("Hi"),
	}, llm.WithExtraField("model", "override-model"))
	if err == nil || !strings.Contains(err.Error(), "conflicts") {
		t.Fatalf("expected conflict error, got: %v", err)
	}

	_, err = c.Chat(context.Background(), []schema.Message{
		schema.UserMessage("Hi"),
	}, llm.WithAllowExtraFieldOverride(true), llm.WithExtraField("model", "override-model"))
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if gotReq["model"] != "override-model" {
		t.Fatalf("model override: got %#v", gotReq["model"])
	}
}
