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
		switch r.URL.Path {
		case "/v1/users/1":
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"id":1}`)
		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = io.WriteString(w, `{"error":"not found"}`)
		}
	}))
	defer srv.Close()

	client, err := httpx.New(
		httpx.WithBaseURL(srv.URL),
		httpx.WithTimeout(3*time.Second),
	)
	if err != nil {
		panic(err)
	}

	req, err := client.NewRequest(context.Background(), http.MethodGet, "/v1/users/1")
	if err != nil {
		panic(err)
	}

	var out struct {
		ID int `json:"id"`
	}
	resp, err := client.DoJSONInto(req, &out)
	if err != nil {
		panic(err)
	}
	_ = resp

	fmt.Println("user.id =", out.ID)
}
