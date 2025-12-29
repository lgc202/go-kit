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

	finishReason llm.FinishReason
	pending      []llm.StreamEvent
}

func newStream(provider string, resp *http.Response) *stream {
	return &stream{
		provider: provider,
		resp:     resp,
		dec:      newSSEDecoder(resp.Body),
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
			return llm.StreamEvent{Kind: llm.StreamEventDone, FinishReason: s.finishReason}, nil
		}
		return llm.StreamEvent{}, err
	}

	data = bytes.TrimSpace(data)
	if bytes.Equal(data, []byte("[DONE]")) {
		s.done = true
		return llm.StreamEvent{Kind: llm.StreamEventDone, FinishReason: s.finishReason, RawJSON: append([]byte(nil), data...)}, nil
	}

	var chunk chatCompletionChunk
	if err := json.Unmarshal(data, &chunk); err != nil {
		return llm.StreamEvent{}, &llm.LLMError{Provider: s.provider, Kind: llm.ErrKindParse, Message: "failed to decode stream chunk", Raw: append([]byte(nil), data...), Cause: err}
	}
	if chunk.Error != nil {
		return llm.StreamEvent{}, &llm.LLMError{Provider: s.provider, Kind: llm.ErrKindServer, Message: chunk.Error.Message, Raw: append([]byte(nil), data...)}
	}

	if chunk.Usage != nil {
		s.pending = append(s.pending, llm.StreamEvent{
			Kind: llm.StreamEventUsage,
			Usage: &llm.Usage{
				PromptTokens:     chunk.Usage.PromptTokens,
				CompletionTokens: chunk.Usage.CompletionTokens,
				TotalTokens:      chunk.Usage.TotalTokens,
			},
			RawJSON: append([]byte(nil), data...),
		})
	}

	for _, choice := range chunk.Choices {
		if choice.FinishReason != "" {
			s.finishReason = mapFinishReason(choice.FinishReason)
		}
		if choice.Delta.ReasoningContent != "" {
			s.pending = append(s.pending, llm.StreamEvent{
				Kind:           llm.StreamEventReasoningDelta,
				ReasoningDelta: choice.Delta.ReasoningContent,
				RawJSON:        append([]byte(nil), data...),
			})
		}
		if thinking := anyString(choice.Delta.Thinking); thinking != "" {
			s.pending = append(s.pending, llm.StreamEvent{
				Kind:           llm.StreamEventReasoningDelta,
				ReasoningDelta: thinking,
				RawJSON:        append([]byte(nil), data...),
			})
		}
		if text := contentText(choice.Delta.Content); text != "" {
			s.pending = append(s.pending, llm.StreamEvent{
				Kind:      llm.StreamEventTextDelta,
				TextDelta: text,
				RawJSON:   append([]byte(nil), data...),
			})
		}
		if _, reasoning := splitContent(choice.Delta.Content); reasoning != "" {
			s.pending = append(s.pending, llm.StreamEvent{
				Kind:           llm.StreamEventReasoningDelta,
				ReasoningDelta: reasoning,
				RawJSON:        append([]byte(nil), data...),
			})
		}
		for _, tc := range choice.Delta.ToolCalls {
			s.pending = append(s.pending, llm.StreamEvent{
				Kind: llm.StreamEventToolCallDelta,
				ToolCallDelta: &llm.ToolCallDelta{
					Index:          tc.Index,
					ID:             tc.ID,
					Name:           tc.Function.Name,
					ArgumentsDelta: tc.Function.Arguments,
				},
				RawJSON: append([]byte(nil), data...),
			})
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
