package ollama

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

func TestOllama_DefaultPathAndProviderName(t *testing.T) {
	httpClient := &http.Client{Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("path=%q", r.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusBadRequest,
			Body:       io.NopCloser(strings.NewReader(`{"error":{"message":"bad request"}}`)),
			Header:     make(http.Header),
			Request:    r,
		}, nil
	})}

	p, err := New("", WithHTTPClient(httpClient))
	if err != nil {
		t.Fatalf("New() err=%v", err)
	}

	client := llm.New(p)
	_, err = client.Chat(context.Background(), []llm.Message{{Role: llm.RoleUser, Content: "hi"}}, llm.WithModel("llama3"))
	if err == nil {
		t.Fatalf("expected error")
	}
	llme, ok := llm.AsLLMError(err)
	if !ok {
		t.Fatalf("expected LLMError, got %T", err)
	}
	if llme.Provider != "ollama" {
		t.Fatalf("Provider=%q", llme.Provider)
	}
}
