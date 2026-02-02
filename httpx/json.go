package httpx

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

func (c *Client) NewJSONRequest(ctx context.Context, method, path string, body any, opts ...RequestOption) (*http.Request, error) {
	opts2 := make([]RequestOption, 0, len(opts)+2)
	opts2 = append(opts2, WithJSON(body))
	opts2 = append(opts2, opts...)
	req, err := c.NewRequest(ctx, method, path, opts2...)
	if err != nil {
		return nil, err
	}
	if req.Header.Get("Accept") == "" {
		req.Header.Set("Accept", "application/json")
	}
	return req, nil
}

// DoJSONInto performs the request, treats non-2xx as error, and decodes a JSON response into dst.
// The response body is always closed.
func (c *Client) DoJSONInto(req *http.Request, dst any) (*http.Response, error) {
	resp, err := c.DoStatus(req)
	if err != nil {
		return resp, err
	}
	if resp == nil || resp.Body == nil {
		return resp, errors.New("nil response body")
	}
	defer resp.Body.Close()

	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(dst); err != nil {
		return resp, err
	}
	// Ensure there's no extra non-whitespace payload.
	var extra any
	if err := dec.Decode(&extra); err != nil && !errors.Is(err, io.EOF) {
		return resp, err
	}
	if extra != nil {
		return resp, errors.New("unexpected extra JSON value in response body")
	}
	return resp, nil
}

// DoJSONIntoStrict is like DoJSONInto but rejects unknown fields.
// Use this when you want "contract" enforcement between client and server.
func (c *Client) DoJSONIntoStrict(req *http.Request, dst any) (*http.Response, error) {
	resp, err := c.DoStatus(req)
	if err != nil {
		return resp, err
	}
	if resp == nil || resp.Body == nil {
		return resp, errors.New("nil response body")
	}
	defer resp.Body.Close()

	dec := json.NewDecoder(resp.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return resp, err
	}
	// Ensure there's no extra non-whitespace payload.
	var extra any
	if err := dec.Decode(&extra); err != nil && !errors.Is(err, io.EOF) {
		return resp, err
	}
	if extra != nil {
		return resp, errors.New("unexpected extra JSON value in response body")
	}
	return resp, nil
}

// DoJSON is a generic helper around DoJSONInto.
func DoJSON[T any](c *Client, req *http.Request) (T, *http.Response, error) {
	var out T
	resp, err := c.DoJSONInto(req, &out)
	if err != nil {
		var zero T
		return zero, resp, err
	}
	return out, resp, nil
}
