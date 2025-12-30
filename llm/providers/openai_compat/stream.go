package openai_compat

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/lgc202/go-kit/llm"
)

type stream struct {
	provider string
	resp     *http.Response
	dec      *sseDecoder

	closed bool
	done   bool

	includeRaw bool

	finishReasons map[int]llm.FinishReason
	pending       []llm.StreamEvent
}

func newStream(provider string, resp *http.Response, includeRaw bool) *stream {
	return &stream{
		provider:   provider,
		resp:       resp,
		dec:        newSSEDecoder(resp.Body),
		includeRaw: includeRaw,
	}
}

func (s *stream) Close() error {
	if s.closed {
		return nil
	}
	s.closed = true
	if s.resp != nil && s.resp.Body != nil {
		return s.resp.Body.Close()
	}
	return nil
}

func (s *stream) Recv() (llm.StreamEvent, error) {
	if s.closed {
		return llm.StreamEvent{}, llm.ErrStreamClosed
	}
	if len(s.pending) > 0 {
		ev := s.pending[0]
		s.pending = s.pending[1:]
		return ev, nil
	}
	if s.done {
		return llm.StreamEvent{}, io.EOF
	}

	data, err := s.dec.Next()
	if err != nil {
		if errors.Is(err, io.EOF) {
			// Some providers close the connection without sending [DONE].
			s.done = true
			return llm.StreamEvent{Kind: llm.StreamEventDone, ChoiceIndex: -1}, nil
		}
		return llm.StreamEvent{}, err
	}

	data = bytes.TrimSpace(data)
	if bytes.Equal(data, []byte("[DONE]")) {
		s.done = true
		ev := llm.StreamEvent{Kind: llm.StreamEventDone, ChoiceIndex: -1}
		if s.includeRaw {
			ev.RawJSON = append([]byte(nil), data...)
		}
		return ev, nil
	}

	var chunk chatCompletionChunk
	if err := json.Unmarshal(data, &chunk); err != nil {
		return llm.StreamEvent{}, &llm.LLMError{Provider: s.provider, Kind: llm.ErrKindParse, Message: "failed to decode stream chunk", Raw: append([]byte(nil), data...), Cause: err}
	}
	if chunk.Error != nil {
		return llm.StreamEvent{}, &llm.LLMError{Provider: s.provider, Kind: llm.ErrKindServer, Message: chunk.Error.Message, Raw: append([]byte(nil), data...)}
	}

	if chunk.Usage != nil {
		var details *llm.UsageDetails
		d := llm.UsageDetails{
			PromptCacheHitTokens:  chunk.Usage.intField("prompt_cache_hit_tokens"),
			PromptCacheMissTokens: chunk.Usage.intField("prompt_cache_miss_tokens"),
		}
		cachedTokens := chunk.Usage.intField("cached_tokens")
		if d.PromptCacheHitTokens == 0 && cachedTokens != 0 {
			d.PromptCacheHitTokens = cachedTokens
		}
		d.ReasoningTokens = chunk.Usage.intFieldInObject("completion_tokens_details", "reasoning_tokens")
		if d.PromptCacheHitTokens != 0 || d.PromptCacheMissTokens != 0 || d.ReasoningTokens != 0 {
			details = &d
		}
		ev := llm.StreamEvent{
			Kind:        llm.StreamEventUsage,
			ChoiceIndex: -1,
			Usage: &llm.Usage{
				PromptTokens:     chunk.Usage.PromptTokens,
				CompletionTokens: chunk.Usage.CompletionTokens,
				TotalTokens:      chunk.Usage.TotalTokens,
				Details:          details,
			},
		}
		if s.includeRaw {
			ev.RawJSON = append([]byte(nil), data...)
		}
		s.pending = append(s.pending, ev)
	}

	for _, choice := range chunk.Choices {
		choiceIdx := choice.Index
		if choice.FinishReason != "" {
			if s.finishReasons == nil {
				s.finishReasons = make(map[int]llm.FinishReason)
			}
			s.finishReasons[choiceIdx] = mapFinishReason(choice.FinishReason)
			ev := llm.StreamEvent{
				Kind:         llm.StreamEventChoiceDone,
				ChoiceIndex:  choiceIdx,
				FinishReason: s.finishReasons[choiceIdx],
			}
			if s.includeRaw {
				ev.RawJSON = append([]byte(nil), data...)
			}
			s.pending = append(s.pending, ev)
		}
		if choice.Delta.ReasoningContent != "" {
			ev := llm.StreamEvent{
				Kind:        llm.StreamEventPartDelta,
				ChoiceIndex: choiceIdx,
				PartDelta: &llm.PartDelta{
					Type:      llm.ContentPartReasoning,
					TextDelta: choice.Delta.ReasoningContent,
				},
			}
			if s.includeRaw {
				ev.RawJSON = append([]byte(nil), data...)
			}
			s.pending = append(s.pending, ev)
		}
		if thinking := anyString(choice.Delta.Thinking); thinking != "" {
			ev := llm.StreamEvent{
				Kind:        llm.StreamEventPartDelta,
				ChoiceIndex: choiceIdx,
				PartDelta: &llm.PartDelta{
					Type:      llm.ContentPartReasoning,
					TextDelta: thinking,
				},
			}
			if s.includeRaw {
				ev.RawJSON = append([]byte(nil), data...)
			}
			s.pending = append(s.pending, ev)
		}
		text, reasoning := splitContent(choice.Delta.Content)
		if text != "" {
			ev := llm.StreamEvent{
				Kind:        llm.StreamEventPartDelta,
				ChoiceIndex: choiceIdx,
				PartDelta: &llm.PartDelta{
					Type:      llm.ContentPartText,
					TextDelta: text,
				},
			}
			if s.includeRaw {
				ev.RawJSON = append([]byte(nil), data...)
			}
			s.pending = append(s.pending, ev)
		}
		if reasoning != "" {
			ev := llm.StreamEvent{
				Kind:        llm.StreamEventPartDelta,
				ChoiceIndex: choiceIdx,
				PartDelta: &llm.PartDelta{
					Type:      llm.ContentPartReasoning,
					TextDelta: reasoning,
				},
			}
			if s.includeRaw {
				ev.RawJSON = append([]byte(nil), data...)
			}
			s.pending = append(s.pending, ev)
		}
		for _, tc := range choice.Delta.ToolCalls {
			ev := llm.StreamEvent{
				Kind:        llm.StreamEventToolCallDelta,
				ChoiceIndex: choiceIdx,
				ToolCallDelta: &llm.ToolCallDelta{
					Index:          tc.Index,
					ID:             tc.ID,
					Name:           tc.Function.Name,
					ArgumentsDelta: tc.Function.Arguments,
				},
			}
			if s.includeRaw {
				ev.RawJSON = append([]byte(nil), data...)
			}
			s.pending = append(s.pending, ev)
		}
	}

	if len(s.pending) == 0 {
		// Nothing meaningful in this chunk; read the next one.
		return s.Recv()
	}

	ev := s.pending[0]
	s.pending = s.pending[1:]
	return ev, nil
}
