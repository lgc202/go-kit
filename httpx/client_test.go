package httpx

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestResolveURL_BaseURLAndQuery(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(r.URL.Path + "?" + r.URL.RawQuery))
	}))
	t.Cleanup(srv.Close)

	c, err := New(
		WithBaseURL(srv.URL),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	req, err := c.NewRequest(context.Background(), http.MethodGet, "/v1/test?x=1",
		WithQueryParam("y", "2"),
	)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	resp, err := c.Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	b, _ := io.ReadAll(resp.Body)
	got := string(b)
	if !strings.HasPrefix(got, "/v1/test?") || !strings.Contains(got, "x=1") || !strings.Contains(got, "y=2") {
		t.Fatalf("unexpected path/query: %q", got)
	}
}

func TestResolveURL_BaseURLWithPathPrefix(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	t.Cleanup(srv.Close)

	c, err := New(WithBaseURL(srv.URL + "/api/v1"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	req, err := c.NewRequest(context.Background(), http.MethodGet, "/users")
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	resp, err := c.Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	_ = resp.Body.Close()

	if gotPath != "/api/v1/users" {
		t.Fatalf("unexpected path: %q", gotPath)
	}
}

func TestDoStatus_RetriesOn5xx(t *testing.T) {
	var n int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c := atomic.AddInt32(&n, 1)
		if c < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("nope"))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	t.Cleanup(srv.Close)

	c, err := New(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	req, err := c.NewRequest(context.Background(), http.MethodGet, "/")
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	resp, err := c.DoStatus(req)
	if err != nil {
		t.Fatalf("DoStatus: %v", err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })

	if got := atomic.LoadInt32(&n); got != 3 {
		t.Fatalf("expected 3 attempts, got %d", got)
	}
}

func TestDoStatus_NoRetryForPOSTByDefault(t *testing.T) {
	var n int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&n, 1)
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("nope"))
	}))
	t.Cleanup(srv.Close)

	c, err := New(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	req, err := c.NewRequest(context.Background(), http.MethodPost, "/",
		WithBodyBytes([]byte(`{}`)),
		WithHeader("Content-Type", "application/json"),
	)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	_, err = c.DoStatus(req)
	if err == nil {
		t.Fatalf("expected error")
	}
	if got := atomic.LoadInt32(&n); got != 1 {
		t.Fatalf("expected 1 attempt, got %d", got)
	}
}

func TestDoStatus_ErrorBodyLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(strings.Repeat("a", 100)))
	}))
	t.Cleanup(srv.Close)

	c, err := New(
		WithBaseURL(srv.URL),
		WithMaxErrorBodyBytes(10),
		WithRetry(RetryConfig{MaxAttempts: 1}),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	req, err := c.NewRequest(context.Background(), http.MethodGet, "/")
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	resp, err := c.DoStatus(req)
	if err == nil {
		t.Fatalf("expected error")
	}
	he, ok := AsError(err)
	if !ok {
		t.Fatalf("expected *httpx.Error, got %T", err)
	}
	if len(he.RawBody) != 10 {
		t.Fatalf("expected RawBody len=10, got %d", len(he.RawBody))
	}
	b, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if len(b) != 10 {
		t.Fatalf("expected resp.Body len=10, got %d", len(b))
	}
}

func TestRequestTimeoutOption(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	t.Cleanup(srv.Close)

	c, err := New(
		WithBaseURL(srv.URL),
		WithTimeout(2*time.Second),
		WithRetry(RetryConfig{MaxAttempts: 1}),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	req, err := c.NewRequest(context.Background(), http.MethodGet, "/",
		WithRequestTimeout(50*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	_, err = c.DoStatus(req)
	if err == nil {
		t.Fatalf("expected timeout error")
	}
}
