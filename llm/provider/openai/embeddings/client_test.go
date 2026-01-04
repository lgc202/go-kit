package embeddings

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/lgc202/go-kit/llm"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestEmbed_RequestAndResponse(t *testing.T) {
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
  "model":"text-embedding-3-small",
  "data":[{"index":0,"embedding":[0.1,0.2]}],
  "usage":{"prompt_tokens":1,"total_tokens":1}
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
		BaseConfig: BaseConfig{
			BaseURL:    "https://example.test/v1",
			APIKey:     "tok",
			HTTPClient: httpClient,
		},
		DefaultOptions: []llm.EmbeddingOption{llm.WithModel("text-embedding-3-small")},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	resp, err := c.Embed(context.Background(), []string{"Hi"})
	if err != nil {
		t.Fatalf("Embed: %v", err)
	}

	if gotPath != "/v1/embeddings" {
		t.Fatalf("path: got %q", gotPath)
	}
	if gotReq["model"] != "text-embedding-3-small" {
		t.Fatalf("request.model: got %#v", gotReq["model"])
	}
	if _, ok := gotReq["input"]; !ok {
		t.Fatalf("request.input missing: %#v", gotReq)
	}

	if resp.Model != "text-embedding-3-small" {
		t.Fatalf("resp.model: got %q", resp.Model)
	}
	if len(resp.Data) != 1 || len(resp.Data[0].Vector) != 2 {
		t.Fatalf("resp.data: %#v", resp.Data)
	}
	if resp.Usage.PromptTokens != 1 || resp.Usage.TotalTokens != 1 {
		t.Fatalf("resp.usage: %#v", resp.Usage)
	}
}
