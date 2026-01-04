package llm

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// APIError API 错误，用于非 2xx 响应
//
// 设计用于企业级处理：分类（限流、认证）、可观测性（请求追踪）、重试（retry-after）
type APIError struct {
	Provider   Provider
	StatusCode int

	// Code provider 特定的错误码
	Code string

	// Type provider 特定的错误类型
	Type string

	// Message 人类可读的错误消息
	Message string

	// RequestID 请求追踪 ID
	RequestID string

	// RetryAfter 重试等待时间
	RetryAfter time.Duration

	// Raw 原始响应体
	Raw []byte
}

func (e *APIError) Error() string {
	if e == nil {
		return "<nil>"
	}

	var b strings.Builder
	if strings.TrimSpace(string(e.Provider)) != "" {
		b.WriteString(string(e.Provider))
		b.WriteString(": ")
	}
	if e.StatusCode != 0 {
		b.WriteString(fmt.Sprintf("http %d", e.StatusCode))
	} else {
		b.WriteString("http error")
	}

	msg := strings.TrimSpace(e.Message)
	if msg == "" && e.StatusCode != 0 {
		msg = http.StatusText(e.StatusCode)
	}
	if msg != "" {
		b.WriteString(": ")
		b.WriteString(msg)
	}

	if strings.TrimSpace(e.Code) != "" {
		b.WriteString(" (")
		b.WriteString(strings.TrimSpace(e.Code))
		b.WriteString(")")
	}
	if strings.TrimSpace(e.RequestID) != "" {
		b.WriteString(" request_id=")
		b.WriteString(strings.TrimSpace(e.RequestID))
	}

	return b.String()
}

// AsAPIError 判断错误是否为 APIError
func AsAPIError(err error) (*APIError, bool) {
	var ae *APIError
	if errors.As(err, &ae) {
		return ae, true
	}
	return nil, false
}

// IsRateLimit 判断是否为限流错误
func IsRateLimit(err error) bool {
	ae, ok := AsAPIError(err)
	if !ok {
		return false
	}
	if ae.StatusCode == http.StatusTooManyRequests {
		return true
	}
	code := strings.ToLower(strings.TrimSpace(ae.Code))
	return code == "rate_limit" || code == "rate_limit_exceeded"
}

// IsAuth 判断是否为认证错误
func IsAuth(err error) bool {
	ae, ok := AsAPIError(err)
	if !ok {
		return false
	}
	return ae.StatusCode == http.StatusUnauthorized || ae.StatusCode == http.StatusForbidden
}

// IsTemporary 判断是否为临时错误（可重试）
func IsTemporary(err error) bool {
	ae, ok := AsAPIError(err)
	if !ok {
		return false
	}
	switch ae.StatusCode {
	case http.StatusRequestTimeout, http.StatusTooManyRequests:
		return true
	case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}
