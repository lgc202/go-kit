package deepseek

import (
	"github.com/lgc202/go-kit/llm"
	"github.com/lgc202/go-kit/llm/internal/openai_compat"
)

type adapter struct {
	openai_compat.NoopAdapter
}

func (adapter) ApplyRequestExtensions(req map[string]any, cfg llm.RequestConfig) error {
	if cfg.Extensions == nil {
		// Still allow generic compat-layer fields to be used without extensions.
		return nil
	}

	if v, ok := cfg.Extensions[extThinking]; ok {
		req["thinking"] = v
	}
	if v, ok := cfg.Extensions[extResponseFormat]; ok {
		req["response_format"] = v
	}
	if v, ok := cfg.Extensions[extFrequencyPenalty]; ok {
		req["frequency_penalty"] = v
	}
	if v, ok := cfg.Extensions[extPresencePenalty]; ok {
		req["presence_penalty"] = v
	}
	if v, ok := cfg.Extensions[extTools]; ok {
		req["tools"] = v
	}
	if v, ok := cfg.Extensions[extToolChoice]; ok {
		req["tool_choice"] = v
	}
	if v, ok := cfg.Extensions[extLogprobs]; ok {
		req["logprobs"] = v
	}
	if v, ok := cfg.Extensions[extTopLogprobs]; ok {
		req["top_logprobs"] = v
	}
	if v, ok := cfg.Extensions[extStreamOptions]; ok {
		req["stream_options"] = v
	}

	return nil
}
