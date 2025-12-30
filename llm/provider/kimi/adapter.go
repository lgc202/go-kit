package kimi

import "github.com/lgc202/go-kit/llm/internal/openai_compat"

type adapter struct {
	openai_compat.NoopAdapter
}

var _ openai_compat.Adapter = adapter{}
