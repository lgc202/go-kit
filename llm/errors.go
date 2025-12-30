package llm

import "fmt"

// UnsupportedOptionError 表示请求选项或 schema 功能不被给定的提供商/客户端实现支持
//
// 使用 errors.As(err, *UnsupportedOptionError) 来判断此错误
type UnsupportedOptionError struct {
	Provider Provider
	Option   string
	Reason   string
}

func (e *UnsupportedOptionError) Error() string {
	if e == nil {
		return "<nil>"
	}

	p := string(e.Provider)
	if p == "" {
		p = string(ProviderUnknown)
	}

	switch {
	case e.Option != "" && e.Reason != "":
		return fmt.Sprintf("%s: unsupported option %q: %s", p, e.Option, e.Reason)
	case e.Option != "":
		return fmt.Sprintf("%s: unsupported option %q", p, e.Option)
	case e.Reason != "":
		return fmt.Sprintf("%s: unsupported option: %s", p, e.Reason)
	default:
		return fmt.Sprintf("%s: unsupported option", p)
	}
}

// APIError 表示提供商 HTTP 错误，尽可能包含解析后的字段
//
// 提供商通常返回包含 message、type 和 code 的 JSON 响应体。
// 当这些字段存在时，openai_compat 会返回 *APIError，以便调用者可以基于
// StatusCode/Code/Type 进行判断
type APIError struct {
	Provider   string
	StatusCode int

	Message string
	Type    string
	Code    string

	// Body 是非 JSON 或无法识别格式时的原始响应体
	Body []byte
}

func (e *APIError) Error() string {
	if e == nil {
		return "<nil>"
	}

	if e.Message != "" && e.Code != "" {
		return fmt.Sprintf("%s: http %d: %s (%s)", e.Provider, e.StatusCode, e.Message, e.Code)
	}
	if e.Message != "" {
		return fmt.Sprintf("%s: http %d: %s", e.Provider, e.StatusCode, e.Message)
	}
	if len(e.Body) > 0 {
		return fmt.Sprintf("%s: http %d: %s", e.Provider, e.StatusCode, string(e.Body))
	}
	return fmt.Sprintf("%s: http %d", e.Provider, e.StatusCode)
}
