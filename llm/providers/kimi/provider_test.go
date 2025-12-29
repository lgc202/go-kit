package kimi

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

func TestKimi_DefaultPathAndProviderName(t *testing.T) {
	httpClient := &http.Client{Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/v1/chat/completions" {
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

	client := llm.New(p)
	_, err = client.Chat(context.Background(), []llm.Message{{Role: llm.RoleUser, Content: "hi"}}, llm.WithModel("moonshot-v1-8k"))
	if err == nil {
		t.Fatalf("expected error")
	}
	llme, ok := llm.AsLLMError(err)
	if !ok {
		t.Fatalf("expected LLMError, got %T", err)
	}
	if llme.Provider != "kimi" {
		t.Fatalf("Provider=%q", llme.Provider)
	}
}
