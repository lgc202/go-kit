package llm

import (
	"context"
	"net/http"
	"time"

	"github.com/lgc202/go-kit/llm/schema"
)

type RequestOption func(*RequestConfig)

type RequestConfig struct {
	Model       string
	Temperature *float64
	TopP        *float64
	MaxTokens   *int
	Stop        *[]string

	FrequencyPenalty *float64
	PresencePenalty  *float64

	Logprobs    *bool
	TopLogprobs *int

	Tools          []schema.Tool
	ToolChoice     *schema.ToolChoice
	ResponseFormat *schema.ResponseFormat

	Timeout *time.Duration
	Headers http.Header

	ExtraBodyFields map[string]any

	// KeepRaw retains provider-native raw JSON payloads in schema.ChatResponse.Raw
	// and schema.StreamEvent.Raw. Default is false to reduce memory usage.
	KeepRaw bool

	// Streaming callbacks are not sent to providers; they are consumed by
	// llm.Wrap(...).Chat when using the streaming API.
	StreamingFunc          func(ctx context.Context, chunk []byte) error
	StreamingReasoningFunc func(ctx context.Context, reasoningChunk, chunk []byte) error

	// Extensions allows provider-specific per-request settings without
	// expanding the cross-provider config surface.
	Extensions map[string]any
}

func (c RequestConfig) clone() RequestConfig {
	out := c

	if c.Stop != nil {
		cp := append([]string(nil), (*c.Stop)...)
		out.Stop = &cp
	}

	if c.FrequencyPenalty != nil {
		v := *c.FrequencyPenalty
		out.FrequencyPenalty = &v
	}
	if c.PresencePenalty != nil {
		v := *c.PresencePenalty
		out.PresencePenalty = &v
	}
	if c.Logprobs != nil {
		v := *c.Logprobs
		out.Logprobs = &v
	}
	if c.TopLogprobs != nil {
		v := *c.TopLogprobs
		out.TopLogprobs = &v
	}

	if c.Tools != nil {
		out.Tools = append([]schema.Tool(nil), c.Tools...)
	}
	if c.ToolChoice != nil {
		v := *c.ToolChoice
		out.ToolChoice = &v
	}
	if c.ResponseFormat != nil {
		v := *c.ResponseFormat
		out.ResponseFormat = &v
	}

	if c.Timeout != nil {
		d := *c.Timeout
		out.Timeout = &d
	}

	if c.Headers != nil {
		out.Headers = c.Headers.Clone()
	}

	if c.ExtraBodyFields != nil {
		out.ExtraBodyFields = make(map[string]any, len(c.ExtraBodyFields))
		for k, v := range c.ExtraBodyFields {
			out.ExtraBodyFields[k] = v
		}
	}

	if c.Extensions != nil {
		out.Extensions = make(map[string]any, len(c.Extensions))
		for k, v := range c.Extensions {
			out.Extensions[k] = v
		}
	}

	return out
}

func ApplyRequestOptions(base RequestConfig, opts ...RequestOption) RequestConfig {
	cfg := base.clone()
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		opt(&cfg)
	}
	return cfg
}

func WithModel(model string) RequestOption {
	return func(c *RequestConfig) {
		c.Model = model
	}
}

func WithTemperature(v float64) RequestOption {
	return func(c *RequestConfig) {
		c.Temperature = &v
	}
}

func WithTopP(v float64) RequestOption {
	return func(c *RequestConfig) {
		c.TopP = &v
	}
}

func WithMaxTokens(v int) RequestOption {
	return func(c *RequestConfig) {
		c.MaxTokens = &v
	}
}

func WithStop(stop ...string) RequestOption {
	return func(c *RequestConfig) {
		cp := append([]string(nil), stop...)
		c.Stop = &cp
	}
}

func WithFrequencyPenalty(v float64) RequestOption {
	return func(c *RequestConfig) {
		c.FrequencyPenalty = &v
	}
}

func WithPresencePenalty(v float64) RequestOption {
	return func(c *RequestConfig) {
		c.PresencePenalty = &v
	}
}

func WithLogprobs(enabled bool) RequestOption {
	return func(c *RequestConfig) {
		c.Logprobs = &enabled
	}
}

func WithTopLogprobs(v int) RequestOption {
	return func(c *RequestConfig) {
		c.TopLogprobs = &v
	}
}

func WithTools(tools ...schema.Tool) RequestOption {
	return func(c *RequestConfig) {
		c.Tools = append([]schema.Tool(nil), tools...)
	}
}

func WithToolChoice(choice schema.ToolChoice) RequestOption {
	return func(c *RequestConfig) {
		v := choice
		c.ToolChoice = &v
	}
}

func WithResponseFormat(format schema.ResponseFormat) RequestOption {
	return func(c *RequestConfig) {
		v := format
		c.ResponseFormat = &v
	}
}

func WithTimeout(d time.Duration) RequestOption {
	return func(c *RequestConfig) {
		c.Timeout = &d
	}
}

func WithHeader(key, value string) RequestOption {
	return func(c *RequestConfig) {
		if c.Headers == nil {
			c.Headers = make(http.Header)
		}
		c.Headers.Set(key, value)
	}
}

func WithExtraHeaders(headers map[string]string) RequestOption {
	return func(c *RequestConfig) {
		if len(headers) == 0 {
			return
		}
		if c.Headers == nil {
			c.Headers = make(http.Header)
		}
		for k, v := range headers {
			c.Headers.Set(k, v)
		}
	}
}

func WithExtraBodyFields(fields map[string]any) RequestOption {
	return func(c *RequestConfig) {
		if len(fields) == 0 {
			return
		}
		if c.ExtraBodyFields == nil {
			c.ExtraBodyFields = make(map[string]any, len(fields))
		}
		for k, v := range fields {
			c.ExtraBodyFields[k] = v
		}
	}
}

func WithExtension(key string, value any) RequestOption {
	return func(c *RequestConfig) {
		if c.Extensions == nil {
			c.Extensions = make(map[string]any)
		}
		c.Extensions[key] = value
	}
}

func WithKeepRaw(enabled bool) RequestOption {
	return func(c *RequestConfig) {
		c.KeepRaw = enabled
	}
}

func WithStreamingFunc(f func(ctx context.Context, chunk []byte) error) RequestOption {
	return func(c *RequestConfig) {
		c.StreamingFunc = f
	}
}

func WithStreamingReasoningFunc(f func(ctx context.Context, reasoningChunk, chunk []byte) error) RequestOption {
	return func(c *RequestConfig) {
		c.StreamingReasoningFunc = f
	}
}
