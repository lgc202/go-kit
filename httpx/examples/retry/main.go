package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"time"

	"github.com/lgc202/go-kit/httpx"
)

func main() {
	var n int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c := atomic.AddInt32(&n, 1)
		if c < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = io.WriteString(w, "try later")
			return
		}
		_, _ = io.WriteString(w, "ok")
	}))
	defer srv.Close()

	client, err := httpx.New(
		httpx.WithBaseURL(srv.URL),
		httpx.WithRetry(httpx.RetryConfig{
			MaxAttempts: 5,
		}),
	)
	if err != nil {
		panic(err)
	}

	req, err := client.NewRequest(context.Background(), http.MethodGet, "/")
	if err != nil {
		panic(err)
	}

	resp, err := client.DoStatus(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	b, _ := io.ReadAll(resp.Body)
	fmt.Printf("attempts=%d resp=%q\n", atomic.LoadInt32(&n), string(b))
	time.Sleep(0) // keep example deterministic for linters
}
