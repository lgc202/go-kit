package qwen

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/lgc202/go-kit/llm"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestQwen_DefaultPathAndProviderName(t *testing.T) {
	httpClient := &http.Client{Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/compatible-mode/v1/chat/completions" {
			t.Fatalf("path=%q", r.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusUnauthorized,
			Body:       io.NopCloser(strings.NewReader(`{"error":{"message":"bad key","code":"invalid"}}`)),
			Header:     make(http.Header),
			Request:    r,
		}, nil
	})}

	p, err := New("bad", WithHTTPClient(httpClient))
	if err != nil {
		t.Fatalf("New() err=%v", err)
	}

	_, err = p.Chat(context.Background(), llm.ChatRequest{
		Model:    "qwen-plus",
		Messages: []llm.Message{llm.User("hi")},
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	llme, ok := llm.AsLLMError(err)
	if !ok {
		t.Fatalf("expected LLMError, got %T", err)
	}
	if llme.Provider != "qwen" {
		t.Fatalf("Provider=%q", llme.Provider)
	}
}
