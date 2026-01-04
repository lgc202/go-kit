package transport

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/lgc202/go-kit/llm"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestClient_PostJSON_Non2xxReturnsAPIError(t *testing.T) {
	t.Parallel()

	httpClient := &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			h := make(http.Header)
			h.Set("Content-Type", "application/json")
			h.Set("Retry-After", "2")
			h.Set("X-Request-Id", "rid_123")
			body := `{"error":{"message":"rate limited","type":"rate_limit_error","code":"rate_limit_exceeded"}}`
			return &http.Response{
				StatusCode: http.StatusTooManyRequests,
				Header:     h,
				Body:       io.NopCloser(strings.NewReader(body)),
				Request:    r,
			}, nil
		}),
	}

	c, err := New(Config{
		Provider:    llm.ProviderOpenAI,
		BaseURL:     "https://example.com/v1",
		DefaultPath: "/chat/completions",
		HTTPClient:  httpClient,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, err = c.PostJSON(context.Background(), map[string]any{"x": 1}, RequestConfig{}, "")
	if err == nil {
		t.Fatalf("expected error")
	}

	ae, ok := llm.AsAPIError(err)
	if !ok {
		t.Fatalf("expected *llm.APIError, got %T: %v", err, err)
	}
	if ae.Provider != llm.ProviderOpenAI {
		t.Fatalf("Provider: got %q", ae.Provider)
	}
	if ae.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("StatusCode: got %d", ae.StatusCode)
	}
	if ae.Code != "rate_limit_exceeded" {
		t.Fatalf("Code: got %q", ae.Code)
	}
	if ae.Type != "rate_limit_error" {
		t.Fatalf("Type: got %q", ae.Type)
	}
	if ae.Message != "rate limited" {
		t.Fatalf("Message: got %q", ae.Message)
	}
	if ae.RequestID != "rid_123" {
		t.Fatalf("RequestID: got %q", ae.RequestID)
	}
	if ae.RetryAfter != 2*time.Second {
		t.Fatalf("RetryAfter: got %v", ae.RetryAfter)
	}
	if len(ae.Raw) == 0 {
		t.Fatalf("Raw: expected non-empty")
	}
	if !llm.IsRateLimit(err) {
		t.Fatalf("IsRateLimit: expected true")
	}
	if llm.IsAuth(err) {
		t.Fatalf("IsAuth: expected false")
	}
	if !llm.IsTemporary(err) {
		t.Fatalf("IsTemporary: expected true")
	}
}

func TestClient_PostJSON_ErrorHookOverride(t *testing.T) {
	t.Parallel()

	httpClient := &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			h := make(http.Header)
			h.Set("Content-Type", "application/json")
			return &http.Response{
				StatusCode: http.StatusBadRequest,
				Header:     h,
				Body:       io.NopCloser(strings.NewReader(`{"error":{"message":"bad","code":"bad"}}`)),
				Request:    r,
			}, nil
		}),
	}

	c, err := New(Config{
		Provider:    llm.ProviderOpenAI,
		BaseURL:     "https://example.com/v1",
		DefaultPath: "/chat/completions",
		HTTPClient:  httpClient,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	hookErr := io.EOF
	_, err = c.PostJSON(
		context.Background(),
		map[string]any{"x": 1},
		RequestConfig{
			ErrorHooks: []llm.ErrorHook{
				func(provider llm.Provider, statusCode int, body []byte) error {
					return hookErr
				},
			},
		},
		"",
	)
	if err == nil {
		t.Fatalf("expected error")
	}
	if err != hookErr {
		t.Fatalf("expected hook error, got %T: %v", err, err)
	}
}

func TestSanitizeHTTPError_PreservesSentinelsAndTypes(t *testing.T) {
	t.Parallel()

	if err := sanitizeHTTPError(context.DeadlineExceeded); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("deadline: expected errors.Is(..., context.DeadlineExceeded) to be true, got %v", err)
	}
	if err := sanitizeHTTPError(context.Canceled); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled: expected errors.Is(..., context.Canceled) to be true, got %v", err)
	}

	timeoutErr := &net.DNSError{IsTimeout: true, Err: "timeout"}
	err := sanitizeHTTPError(timeoutErr)
	if !errors.As(err, new(net.Error)) {
		t.Fatalf("net timeout: expected errors.As(..., net.Error) to be true, got %T: %v", err, err)
	}
	if !errors.Is(err, timeoutErr) {
		t.Fatalf("net timeout: expected wrapped original error")
	}

	netErr := &net.DNSError{IsTimeout: false, Err: "network"}
	err = sanitizeHTTPError(netErr)
	if !errors.As(err, new(net.Error)) {
		t.Fatalf("net: expected errors.As(..., net.Error) to be true, got %T: %v", err, err)
	}
	if !errors.Is(err, netErr) {
		t.Fatalf("net: expected wrapped original error")
	}
}
