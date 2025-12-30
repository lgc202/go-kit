package openai_compat

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/lgc202/go-kit/llm"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestChatStream_TextDelta(t *testing.T) {
	httpClient := &http.Client{Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/v1/chat/completions" {
			return &http.Response{StatusCode: http.StatusNotFound, Body: io.NopCloser(strings.NewReader("")), Header: make(http.Header), Request: r}, nil
		}
		if got := r.Header.Get("Authorization"); !strings.HasPrefix(got, "Bearer ") {
			return &http.Response{StatusCode: http.StatusUnauthorized, Body: io.NopCloser(strings.NewReader("")), Header: make(http.Header), Request: r}, nil
		}
		body, _ := io.ReadAll(r.Body)
		if !bytes.Contains(body, []byte("\"stream\":true")) {
			return &http.Response{StatusCode: http.StatusBadRequest, Body: io.NopCloser(strings.NewReader(`{"error":{"message":"expected stream"}}`)), Header: make(http.Header), Request: r}, nil
		}

		payload := strings.Join([]string{
			"data: " + `{"id":"s1","object":"chat.completion.chunk","created":1,"model":"m","choices":[{"index":0,"delta":{"content":"Hello"},"finish_reason":""}]}`,
			"",
			"data: " + `{"id":"s1","object":"chat.completion.chunk","created":1,"model":"m","choices":[{"index":0,"delta":{"content":" world"},"finish_reason":""}]}`,
			"",
			"data: " + `{"id":"s1","object":"chat.completion.chunk","created":1,"model":"m","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`,
			"",
			"data: [DONE]",
			"",
		}, "\n")

		h := make(http.Header)
		h.Set("Content-Type", "text/event-stream")
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(payload)), Header: h, Request: r}, nil
	})}

	p, err := New("test-key",
		WithProviderName("test"),
		WithBaseURL("https://example.test"),
		WithHTTPClient(httpClient),
		WithDefaultModel("m"),
	)
	if err != nil {
		t.Fatalf("New() err=%v", err)
	}

	stream, err := p.ChatStream(context.Background(), llm.ChatRequest{
		// exercise default model selection
		Messages: []llm.Message{llm.User("hi")},
	})
	if err != nil {
		t.Fatalf("ChatStream() err=%v", err)
	}

	resp, err := llm.DrainStream(stream)
	if err != nil {
		t.Fatalf("DrainStream() err=%v", err)
	}
	if got := resp.FirstText(); got != "Hello world" {
		t.Fatalf("FirstText()=%q", got)
	}
	if resp.Choices[0].FinishReason != llm.FinishReasonStop {
		t.Fatalf("FinishReason=%q", resp.Choices[0].FinishReason)
	}
}

func TestChatStream_ToolCallDelta(t *testing.T) {
	httpClient := &http.Client{Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		payload := strings.Join([]string{
			"data: " + `{"id":"s1","object":"chat.completion.chunk","created":1,"model":"m","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"call_1","type":"function","function":{"name":"get_weather","arguments":"{\"location\":\""}}]},"finish_reason":""}]}`,
			"",
			"data: " + `{"id":"s1","object":"chat.completion.chunk","created":1,"model":"m","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"SF\"}"}}]},"finish_reason":""}]}`,
			"",
			"data: " + `{"id":"s1","object":"chat.completion.chunk","created":1,"model":"m","choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}]}`,
			"",
			"data: [DONE]",
			"",
		}, "\n")

		h := make(http.Header)
		h.Set("Content-Type", "text/event-stream")
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(payload)), Header: h, Request: r}, nil
	})}

	p, err := New("test-key",
		WithProviderName("test"),
		WithBaseURL("https://example.test"),
		WithHTTPClient(httpClient),
		WithDefaultModel("m"),
	)
	if err != nil {
		t.Fatalf("New() err=%v", err)
	}

	stream, err := p.ChatStream(context.Background(), llm.ChatRequest{
		Messages: []llm.Message{llm.User("hi")},
	})
	if err != nil {
		t.Fatalf("ChatStream() err=%v", err)
	}

	var acc llm.Accumulator
	for {
		ev, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			t.Fatalf("Recv() err=%v", err)
		}
		acc.Apply(ev)
		if ev.Done() {
			break
		}
	}
	_ = stream.Close()

	resp := acc.FinalResponse()
	if len(resp.Choices) != 1 {
		t.Fatalf("Choices=%d", len(resp.Choices))
	}
	msg := resp.Choices[0].Message
	if len(msg.ToolCalls) != 1 {
		t.Fatalf("ToolCalls=%d", len(msg.ToolCalls))
	}
	if msg.ToolCalls[0].Name != "get_weather" {
		t.Fatalf("ToolCall.Name=%q", msg.ToolCalls[0].Name)
	}
	if got := msg.ToolCalls[0].ArgumentsText; got != `{"location":"SF"}` {
		t.Fatalf("ArgumentsText=%q", got)
	}
	if !json.Valid(msg.ToolCalls[0].Arguments) {
		t.Fatalf("Arguments should be valid json: %q", string(msg.ToolCalls[0].Arguments))
	}
}

func TestChatStream_ReasoningDelta(t *testing.T) {
	httpClient := &http.Client{Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		payload := strings.Join([]string{
			"data: " + `{"id":"s1","object":"chat.completion.chunk","created":1,"model":"m","choices":[{"index":0,"delta":{"reasoning_content":"Think..."},"finish_reason":""}]}`,
			"",
			"data: " + `{"id":"s1","object":"chat.completion.chunk","created":1,"model":"m","choices":[{"index":0,"delta":{"content":"Answer"},"finish_reason":"stop"}]}`,
			"",
			"data: [DONE]",
			"",
		}, "\n")

		h := make(http.Header)
		h.Set("Content-Type", "text/event-stream")
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(payload)), Header: h, Request: r}, nil
	})}

	p, err := New("test-key",
		WithProviderName("test"),
		WithBaseURL("https://example.test"),
		WithHTTPClient(httpClient),
		WithDefaultModel("m"),
	)
	if err != nil {
		t.Fatalf("New() err=%v", err)
	}

	stream, err := p.ChatStream(context.Background(), llm.ChatRequest{
		Messages: []llm.Message{llm.User("hi")},
	})
	if err != nil {
		t.Fatalf("ChatStream() err=%v", err)
	}
	defer stream.Close()

	var acc llm.Accumulator
	for {
		ev, err := stream.Recv()
		if err != nil {
			break
		}
		acc.Apply(ev)
		if ev.Done() {
			break
		}
	}

	resp := acc.FinalResponse()
	if len(resp.Choices) != 1 {
		t.Fatalf("Choices=%d", len(resp.Choices))
	}
	msg := resp.Choices[0].Message
	if msg.Reasoning() != "Think..." {
		t.Fatalf("Reasoning=%q", msg.Reasoning())
	}
	if msg.Text() != "Answer" {
		t.Fatalf("Content=%q", msg.Text())
	}
}

func TestChat_HTTPErrorMapping(t *testing.T) {
	httpClient := &http.Client{Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusUnauthorized,
			Body:       io.NopCloser(strings.NewReader(`{"error":{"message":"bad key","type":"invalid_request_error","code":"invalid_api_key"}}`)),
			Header:     make(http.Header),
			Request:    r,
		}, nil
	})}

	p, err := New("bad",
		WithProviderName("test"),
		WithBaseURL("https://example.test"),
		WithHTTPClient(httpClient),
		WithChatCompletionsPath("/"),
	)
	if err != nil {
		t.Fatalf("New() err=%v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err = p.Chat(ctx, llm.ChatRequest{Model: "m", Messages: []llm.Message{llm.User("hi")}})
	if err == nil {
		t.Fatalf("expected error")
	}
	llme, ok := llm.AsLLMError(err)
	if !ok {
		t.Fatalf("expected LLMError, got %T", err)
	}
	if llme.Kind != llm.ErrKindAuth {
		t.Fatalf("Kind=%q", llme.Kind)
	}
	if llme.ProviderCode != "invalid_api_key" {
		t.Fatalf("ProviderCode=%q", llme.ProviderCode)
	}
}

func TestChat_UsageDetailsMapping(t *testing.T) {
	httpClient := &http.Client{Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body: io.NopCloser(strings.NewReader(
				`{"id":"x","model":"m","choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}],` +
					`"usage":{"prompt_tokens":1,"completion_tokens":2,"total_tokens":3,"prompt_cache_hit_tokens":4,"prompt_cache_miss_tokens":5,"completion_tokens_details":{"reasoning_tokens":6}}}`,
			)),
			Header:  make(http.Header),
			Request: r,
		}, nil
	})}

	p, err := New("k",
		WithProviderName("test"),
		WithBaseURL("https://example.test"),
		WithHTTPClient(httpClient),
		WithChatCompletionsPath("/"),
	)
	if err != nil {
		t.Fatalf("New() err=%v", err)
	}

	resp, err := p.Chat(context.Background(), llm.ChatRequest{Model: "m", Messages: []llm.Message{llm.User("hi")}})
	if err != nil {
		t.Fatalf("Chat() err=%v", err)
	}
	if resp.Usage == nil {
		t.Fatalf("missing usage")
	}
	if resp.Usage.TotalTokens != 3 {
		t.Fatalf("total=%d", resp.Usage.TotalTokens)
	}
	if resp.Usage.Details == nil {
		t.Fatalf("missing usage details")
	}
	if resp.Usage.Details.PromptCacheHitTokens != 4 {
		t.Fatalf("hit=%d", resp.Usage.Details.PromptCacheHitTokens)
	}
	if resp.Usage.Details.PromptCacheMissTokens != 5 {
		t.Fatalf("miss=%d", resp.Usage.Details.PromptCacheMissTokens)
	}
	if resp.Usage.Details.ReasoningTokens != 6 {
		t.Fatalf("reasoning=%d", resp.Usage.Details.ReasoningTokens)
	}
}

func TestChat_UsageCachedTokensMapping(t *testing.T) {
	httpClient := &http.Client{Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body: io.NopCloser(strings.NewReader(
				`{"id":"x","model":"m","choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}],` +
					`"usage":{"prompt_tokens":19,"completion_tokens":21,"total_tokens":40,"cached_tokens":10}}`,
			)),
			Header:  make(http.Header),
			Request: r,
		}, nil
	})}

	p, err := New("k",
		WithProviderName("test"),
		WithBaseURL("https://example.test"),
		WithHTTPClient(httpClient),
		WithChatCompletionsPath("/"),
	)
	if err != nil {
		t.Fatalf("New() err=%v", err)
	}

	resp, err := p.Chat(context.Background(), llm.ChatRequest{Model: "m", Messages: []llm.Message{llm.User("hi")}})
	if err != nil {
		t.Fatalf("Chat() err=%v", err)
	}
	if resp.Usage == nil || resp.Usage.Details == nil {
		t.Fatalf("missing usage details: %+v", resp.Usage)
	}
	// cached_tokens maps to PromptCacheHitTokens when prompt_cache_hit_tokens is absent.
	if resp.Usage.Details.PromptCacheHitTokens != 10 {
		t.Fatalf("hit=%d", resp.Usage.Details.PromptCacheHitTokens)
	}
}
