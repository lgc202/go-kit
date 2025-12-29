package openai_compat

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/lgc202/go-kit/llm"
)

func TestRequestMapping_CommonFields(t *testing.T) {
	var gotBody []byte
	httpClient := &http.Client{Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		gotBody, _ = io.ReadAll(r.Body)
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"id":"x","model":"m","choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}]}`)),
			Header:     make(http.Header),
			Request:    r,
		}, nil
	})}

	p, err := New("k",
		WithProviderName("test"),
		WithBaseURL("https://example.test"),
		WithHTTPClient(httpClient),
	)
	if err != nil {
		t.Fatalf("New() err=%v", err)
	}

	seed := int64(42)
	maxTokens := 123
	temp := 0.7
	pres := 0.1
	freq := 0.2
	logprobs := true
	topLogProbs := 5

	_, err = p.Chat(context.Background(), llm.ChatRequest{
		Model:            "m",
		Messages:         []llm.Message{{Role: llm.RoleUser, Content: "hi"}},
		Temperature:      &temp,
		MaxTokens:        &maxTokens,
		Seed:             &seed,
		PresencePenalty:  &pres,
		FrequencyPenalty: &freq,
		Stop:             []string{"\n"},
		ResponseFormat:   &llm.ResponseFormat{Type: llm.ResponseFormatText},
		LogProbs:         &logprobs,
		TopLogProbs:      &topLogProbs,
		StreamOptions:    &llm.StreamOptions{IncludeUsage: true},
	})
	if err != nil {
		t.Fatalf("Chat() err=%v", err)
	}

	if len(gotBody) == 0 {
		t.Fatalf("empty body")
	}

	var m map[string]any
	if err := json.Unmarshal(gotBody, &m); err != nil {
		t.Fatalf("unmarshal body: %v\n%s", err, string(gotBody))
	}

	if m["model"] != "m" {
		t.Fatalf("model=%v", m["model"])
	}
	if m["seed"] != float64(42) {
		t.Fatalf("seed=%v", m["seed"])
	}
	if m["max_tokens"] != float64(123) {
		t.Fatalf("max_tokens=%v", m["max_tokens"])
	}
	if m["temperature"] != 0.7 {
		t.Fatalf("temperature=%v", m["temperature"])
	}
	if m["presence_penalty"] != 0.1 {
		t.Fatalf("presence_penalty=%v", m["presence_penalty"])
	}
	if m["frequency_penalty"] != 0.2 {
		t.Fatalf("frequency_penalty=%v", m["frequency_penalty"])
	}
	if _, ok := m["stop"]; !ok {
		t.Fatalf("missing stop")
	}
	if _, ok := m["response_format"]; !ok {
		t.Fatalf("missing response_format")
	}
	if m["logprobs"] != true {
		t.Fatalf("logprobs=%v", m["logprobs"])
	}
	if m["top_logprobs"] != float64(5) {
		t.Fatalf("top_logprobs=%v", m["top_logprobs"])
	}
	if so, ok := m["stream_options"].(map[string]any); !ok || so["include_usage"] != true {
		t.Fatalf("stream_options=%v", m["stream_options"])
	}
}

func TestRequestMapping_ExplicitNullViaExtra(t *testing.T) {
	var gotBody []byte
	httpClient := &http.Client{Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		gotBody, _ = io.ReadAll(r.Body)
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"id":"x","model":"m","choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}]}`)), Header: make(http.Header), Request: r}, nil
	})}

	p, err := New("k", WithProviderName("test"), WithBaseURL("https://example.test"), WithHTTPClient(httpClient))
	if err != nil {
		t.Fatalf("New() err=%v", err)
	}

	_, err = p.Chat(context.Background(), llm.ChatRequest{
		Model:    "m",
		Messages: []llm.Message{{Role: llm.RoleUser, Content: "hi"}},
		Extra: map[string]any{
			"stop":           nil,
			"stream_options": nil,
		},
	})
	if err != nil {
		t.Fatalf("Chat() err=%v", err)
	}

	if !bytes.Contains(gotBody, []byte(`"stop":null`)) {
		t.Fatalf("body=%s", string(gotBody))
	}
	if !bytes.Contains(gotBody, []byte(`"stream_options":null`)) {
		t.Fatalf("body=%s", string(gotBody))
	}
}
