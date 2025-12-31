package openai_compat

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

func TestClient_BaseURLWithPrefixPath(t *testing.T) {
	t.Parallel()

	var gotURL string
	httpClient := &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			gotURL = r.URL.String()
			body := `{
  "id":"abc",
  "created": 1,
  "model":"m",
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
		Provider:       llm.Provider("test"),
		BaseURL:        "https://dashscope-intl.aliyuncs.com/compatible-mode/v1",
		APIKey:         "tok",
		HTTPClient:     httpClient,
		DefaultOptions: []llm.RequestOption{llm.WithModel("m")},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, err = c.Chat(context.Background(), []schema.Message{schema.UserMessage("Hi")})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if gotURL != "https://dashscope-intl.aliyuncs.com/compatible-mode/v1/chat/completions" {
		t.Fatalf("url: got %q", gotURL)
	}
}

func TestClient_BaseURLIsFullEndpoint(t *testing.T) {
	t.Parallel()

	var gotURL string
	var gotReq map[string]any
	httpClient := &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			gotURL = r.URL.String()
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
  "model":"m",
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
		Provider:       llm.Provider("test"),
		BaseURL:        "https://dashscope-intl.aliyuncs.com/compatible-mode/v1/chat/completions",
		APIKey:         "tok",
		HTTPClient:     httpClient,
		DefaultOptions: []llm.RequestOption{llm.WithModel("m")},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, err = c.Chat(context.Background(), []schema.Message{schema.UserMessage("Hi")})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if gotURL != "https://dashscope-intl.aliyuncs.com/compatible-mode/v1/chat/completions" {
		t.Fatalf("url: got %q", gotURL)
	}
	if gotReq["model"] != "m" {
		t.Fatalf("request: model got %#v", gotReq["model"])
	}
}
