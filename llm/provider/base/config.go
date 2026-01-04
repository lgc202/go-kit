package base

import "net/http"

// Config 是 provider 侧通用的基础配置（BaseURL/APIKey/HTTPClient/DefaultHeaders）。
type Config struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client

	// DefaultHeaders 默认请求头，会被请求级别的 headers 覆盖
	DefaultHeaders http.Header
}
