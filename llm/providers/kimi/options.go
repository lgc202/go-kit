package kimi

import "github.com/lgc202/go-kit/llm/providers/openai_compat"

type Option = openai_compat.Option

var (
	WithBaseURL             = openai_compat.WithBaseURL
	WithHTTPClient          = openai_compat.WithHTTPClient
	WithUserAgent           = openai_compat.WithUserAgent
	WithLogger              = openai_compat.WithLogger
	WithRetry               = openai_compat.WithRetry
	WithDefaultHeader       = openai_compat.WithDefaultHeader
	WithChatCompletionsPath = openai_compat.WithChatCompletionsPath
	WithDefaultModel        = openai_compat.WithDefaultModel
	WithHooks               = openai_compat.WithHooks
)
