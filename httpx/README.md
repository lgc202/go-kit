# httpx

`httpx` 是一个面向服务端场景的 HTTP Client 封装，目标是「企业级默认值 + 可观测/可扩展」，并尽量保持：

- 对业务无侵入：不强绑定日志/指标/trace/熔断等具体实现
- 可控的“默认行为”：尤其是超时、重试、错误体读取上限、连接复用

- 复用连接：提供更稳妥的 `http.Transport` 默认配置（见 `DefaultTransport()`）
- 统一 BaseURL + 默认 Header + User-Agent + RequestID 注入
- 内置重试：指数退避 + jitter，默认仅对幂等方法重试，并支持 `Retry-After`
- 错误模型：`*httpx.Error` 携带 `StatusCode/RequestID/RetryAfter/RawBody(截断)` 等字段
- Hook/中间件：便于接入日志/指标/trace（不强绑定具体依赖）

## 安装

```bash
go get github.com/lgc202/go-kit/httpx
```

## 你应该怎么用（核心心智模型）

- Client 级配置：BaseURL/默认头/超时/重试策略/RequestID/Transport
- Request 级配置：Header/Query/Auth/Body/RequestTimeout
- 发送请求：
  - `Do(req)`：更接近 `net/http`（非 2xx 也会返回 `resp,nil`；但会按策略重试 5xx/429 等）
  - `DoStatus(req)`：非 2xx 直接返回 `*httpx.Error`（并截断保存错误 body）
- JSON：
  - `DoJSONInto(req, &dst)`：Decode JSON 到结构体
  - `DoJSONIntoStrict(req, &dst)`：严格模式（拒绝 unknown fields）
  - `httpx.DoJSON[T](client, req)`：泛型辅助（不是方法）

## 快速开始

```go
client, _ := httpx.New(
	httpx.WithBaseURL("https://api.example.com"),
	httpx.WithTimeout(30*time.Second),
	httpx.WithDefaultHeader("Accept", "application/json"),
)

req, _ := client.NewJSONRequest(ctx, http.MethodPost, "/v1/users", map[string]any{
	"name": "alice",
})

resp, err := client.DoStatus(req) // 非 2xx -> *httpx.Error
if err != nil {
	if he, ok := httpx.AsError(err); ok {
		// he.StatusCode / he.RequestID / he.RetryAfter / he.RawBody ...
	}
}
defer resp.Body.Close()
```

## 场景：JSON API（请求 + 响应）

```go
req, _ := client.NewRequest(ctx, http.MethodGet, "/v1/users/1")
var out struct{ ID int `json:"id"` }
resp, err := client.DoJSONInto(req, &out)
_ = resp
_ = err
```

严格模式（拒绝 unknown fields）：

```go
req, _ := client.NewRequest(ctx, http.MethodGet, "/v1/users/1")
var out struct{ ID int `json:"id"` }
resp, err := client.DoJSONIntoStrict(req, &out)
_ = resp
_ = err
```

## 场景：BaseURL + 路径拼接 + Query

```go
client, _ := httpx.New(httpx.WithBaseURL("https://api.example.com"))

req, _ := client.NewRequest(ctx, http.MethodGet, "/v1/search",
	httpx.WithQueryParam("q", "alice"),
	httpx.WithQueryParam("limit", "10"),
)
resp, err := client.DoStatus(req)
_ = resp
_ = err
```

## 场景：鉴权（Bearer / Basic / 自定义签名）

Bearer Token：

```go
req, _ := client.NewRequest(ctx, http.MethodGet, "/v1/me",
	httpx.WithBearerToken(os.Getenv("API_TOKEN")),
)
```

Basic Auth：

```go
req, _ := client.NewRequest(ctx, http.MethodGet, "/v1/me",
	httpx.WithBasicAuth("user", "pass"),
)
```

自定义签名（推荐用 Middleware；注意：如果要签名 body，确保 `req.GetBody` 可用）：

```go
client.WithMiddleware(func(next http.RoundTripper) http.RoundTripper {
	return httpx.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		// 这里放你们的 HMAC/AKSK/时间戳签名逻辑
		// req.Header.Set("Authorization", "...")
		return next.RoundTrip(req)
	})
})
```

## 场景：超时（context / client / request 三层）

- `ctx` 的 deadline 最优先（最早到期的会生效）
- `httpx.WithTimeout`：Client 级总超时（包含重试）
- `httpx.WithRequestTimeout`：Request 级超时上限（仅此请求）

```go
client, _ := httpx.New(httpx.WithTimeout(10*time.Second))

req, _ := client.NewRequest(ctx, http.MethodGet, "/v1/slow",
	httpx.WithRequestTimeout(300*time.Millisecond),
)
_, err := client.DoStatus(req)
_ = err
```

## 场景：重试（默认策略 + 自定义）

默认重试：

- 仅幂等方法（GET/PUT/DELETE/...，不包含 POST）
- 默认 status：429/408/5xx（常见可重试集合）
- 指数退避 + jitter
- 支持 `Retry-After`（用于 429/503）

自定义重试（比如允许 POST、或者只对 5xx 重试）：

```go
client, _ := httpx.New(httpx.WithRetry(httpx.RetryConfig{
	MaxAttempts: 5,
	Methods: map[string]bool{
		http.MethodPost: true, // 谨慎：确保 body 可重放（用 WithBodyBytes/WithJSON 或设置 req.GetBody）
	},
	StatusCodes: map[int]bool{
		http.StatusTooManyRequests:     true,
		http.StatusServiceUnavailable: true,
		http.StatusBadGateway:         true,
	},
	Backoff: httpx.ExponentialBackoff{
		Base:   200 * time.Millisecond,
		Max:    2 * time.Second,
		Jitter: 0.2,
	},
	RespectRetryAfter: true,
	MaxRetryAfter:     30 * time.Second,
}))
```

重试与请求体（重要）：

- `WithJSON(...)` / `WithBodyBytes(...)`：天然可重试（会设置 `req.GetBody`）
- `WithBody(io.Reader)`：默认不可重试（除非你自己设置 `req.GetBody`）

## 场景：下载/流式读取（大响应）

```go
req, _ := client.NewRequest(ctx, http.MethodGet, "/v1/export")
resp, err := client.DoStatus(req)
if err != nil {
	return err
}
defer resp.Body.Close()

_, err = io.Copy(dstWriter, resp.Body)
return err
```

## 场景：上传（文件/大 body）

```go
f, _ := os.Open("big.bin")
defer f.Close()

// 大 body 通常不建议自动重试；若要重试需自行提供 req.GetBody。
client, _ := httpx.New(httpx.WithRetry(httpx.RetryConfig{MaxAttempts: 1}))

req, _ := client.NewRequest(ctx, http.MethodPut, "/v1/upload",
	httpx.WithBody(f),
	httpx.WithHeader("Content-Type", "application/octet-stream"),
)
resp, err := client.DoStatus(req)
_ = resp
_ = err
```

## 场景：错误处理（统一错误模型）

```go
resp, err := client.DoStatus(req)
if err != nil {
	if he, ok := httpx.AsError(err); ok {
		// he.StatusCode / he.RequestID / he.RetryAfter / he.RawBody / he.Retryable
	}
	return err
}
defer resp.Body.Close()
```

## 可观测 / 扩展

### 1) Hooks（每次 attempt 前后都会触发）

```go
client.WithHooks(
	[]httpx.BeforeHook{
		func(req *http.Request, attempt int) error { return nil },
	},
	[]httpx.AfterHook{
		func(req *http.Request, resp *http.Response, err error, dur time.Duration, attempt int) {},
	},
)
```

典型用法：

- logging：记录 method/url/status/err/duration/request_id/attempt
- metrics：统计请求耗时直方图、错误率、重试次数
- tracing：在 Before/After 里建立 span（或直接用 middleware）

### 2) Middleware（RoundTripper 级别）

```go
client.WithMiddleware(func(next http.RoundTripper) http.RoundTripper {
	return httpx.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		// trace/log/metrics...
		return next.RoundTrip(req)
	})
})
```

## 场景：限流（客户端侧）

`httpx` 提供 `RateLimiter` 接口（`Wait(ctx)`）。你可以用任意实现接入：

```go
type tokenBucket struct{ ch chan struct{} }

func (b *tokenBucket) Wait(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-b.ch:
		return nil
	}
}

// 每 10ms 放 1 个 token
b := &tokenBucket{ch: make(chan struct{}, 1)}
go func() {
	t := time.NewTicker(10 * time.Millisecond)
	defer t.Stop()
	for range t.C {
		select { case b.ch <- struct{}{}: default: }
	}
}()

client.WithRateLimiter(b)
```

## 场景：代理 / TLS / mTLS / 自定义 CA

自定义 Transport（推荐从 `DefaultTransport()` clone 后改）：

```go
t := httpx.DefaultTransport()
t.Proxy = http.ProxyFromEnvironment

client, _ := httpx.New(httpx.WithTransport(t))
```

自定义 CA / mTLS（示意；证书加载细节按你们实际来）：

```go
t := httpx.DefaultTransport()
t.TLSClientConfig = &tls.Config{
	MinVersion: tls.VersionTLS12,
	// RootCAs:  yourCertPool,
	// Certificates: []tls.Certificate{yourClientCert},
}
client, _ := httpx.New(httpx.WithTransport(t))
```

## 场景：熔断/隔离（推荐外接）

`httpx` 不内置熔断器，但支持用 middleware/hook 很自然地集成（示意）：

```go
client.WithMiddleware(func(next http.RoundTripper) http.RoundTripper {
	return httpx.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		// if breaker.Open() { return nil, ErrBreakerOpen }
		return next.RoundTrip(req)
	})
})
```

## 场景：测试（不打真实网络）

- `httptest.NewServer(...)`：端到端模拟服务端
- `httpx.RoundTripperFunc`：直接 stub `RoundTrip`（更单元）

```go
rt := httpx.RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(`{"ok":true}`)),
		Header:     make(http.Header),
		Request:    r,
	}, nil
})
client, _ := httpx.New(httpx.WithTransport(rt))
```

## API 速查

Client：

- `New(...Option) (*Client, error)`
- `NewWithConfig(Config) (*Client, error)`
- `(*Client).NewRequest(ctx, method, path, ...RequestOption) (*http.Request, error)`
- `(*Client).NewJSONRequest(ctx, method, path, body, ...RequestOption) (*http.Request, error)`
- `(*Client).Do(req) (*http.Response, error)`
- `(*Client).DoStatus(req) (*http.Response, error)`
- `(*Client).DoJSONInto(req, dst) (*http.Response, error)`
- `(*Client).DoJSONIntoStrict(req, dst) (*http.Response, error)`
- `DoJSON[T](client, req) (T, *http.Response, error)`

Options：

- `WithBaseURL/WithTimeout/WithTransport/WithDefaultHeader(s)/WithUserAgent/WithRetry/WithRequestID/WithMaxErrorBodyBytes`
- Request：`WithHeader(s)/WithQuery(Param)/WithBearerToken/WithBasicAuth/WithBody/WithBodyBytes/WithJSON/WithRequestTimeout`
