package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/lgc202/go-kit/httpx"
)

func main() {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "ok")
	}))
	defer srv.Close()

	client, err := httpx.New(httpx.WithBaseURL(srv.URL))
	if err != nil {
		panic(err)
	}

	client.WithHooks(
		[]httpx.BeforeHook{
			func(req *http.Request, attempt int) error {
				req.Header.Set("X-Tenant", "tenant-a")
				return nil
			},
		},
		[]httpx.AfterHook{
			func(req *http.Request, resp *http.Response, err error, dur time.Duration, attempt int) {
				code := 0
				if resp != nil {
					code = resp.StatusCode
				}
				fmt.Printf("method=%s url=%s status=%d err=%v dur=%s attempt=%d\n",
					req.Method, req.URL.String(), code, err, dur, attempt)
			},
		},
	)

	req, err := client.NewRequest(context.Background(), http.MethodGet, "/")
	if err != nil {
		panic(err)
	}
	resp, err := client.DoStatus(req)
	if err != nil {
		panic(err)
	}
	_ = resp.Body.Close()
}
