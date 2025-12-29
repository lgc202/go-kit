package deepseek

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

func TestDeepSeek_DefaultPathAndProviderName(t *testing.T) {
	httpClient := &http.Client{Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("path=%q", r.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusUnauthorized,
			Body:       io.NopCloser(strings.NewReader(`{"error":{"message":"bad key","code":"invalid"}}`)),
			Header:     make(http.Header),
			Request:    r,
		}, nil
	})}

	p, err := New("bad",
		WithHTTPClient(httpClient),
		WithBaseURL("https://example.test"),
	)
	if err != nil {
		t.Fatalf("New() err=%v", err)
	}

	_, err = p.Chat(context.Background(), llm.ChatRequest{
		Model:    "deepseek-chat",
		Messages: []llm.Message{{Role: llm.RoleUser, Content: "hi"}},
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	llme, ok := llm.AsLLMError(err)
	if !ok {
		t.Fatalf("expected LLMError, got %T", err)
	}
	if llme.Provider != "deepseek" {
		t.Fatalf("Provider=%q", llme.Provider)
	}
}

func TestDeepSeek_ThinkingDisabled(t *testing.T) {
	httpClient := &http.Client{Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), `"thinking":{"type":"disabled"}`) {
			t.Fatalf("body=%s", string(body))
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"id":"x","model":"m","choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}]}`)),
			Header:     make(http.Header),
			Request:    r,
		}, nil
	})}

	p, err := New("key",
		WithHTTPClient(httpClient),
		WithBaseURL("https://example.test"),
		WithDefaultRequest(WithThinkingDisabled()),
	)
	if err != nil {
		t.Fatalf("New() err=%v", err)
	}

	resp, err := p.Chat(context.Background(), llm.ChatRequest{
		Model:    "deepseek-chat",
		Messages: []llm.Message{{Role: llm.RoleUser, Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("Chat() err=%v", err)
	}
	if resp.FirstText() != "ok" {
		t.Fatalf("FirstText=%q", resp.FirstText())
	}
}
