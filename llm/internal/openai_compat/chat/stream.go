package chat

import (
	"encoding/json"
	"io"
	"slices"

	"github.com/lgc202/go-kit/llm"
	"github.com/lgc202/go-kit/llm/internal/openai_compat/transport"
	"github.com/lgc202/go-kit/llm/schema"
)

type stream struct {
	body io.ReadCloser
	dec  *transport.SSEDecoder

	provider string
	keepRaw  bool
	hooks    []llm.StreamEventHook

	pending []schema.StreamEvent
	done    bool
}

const sseDoneToken = "[DONE]"

func newStream(provider string, body io.ReadCloser, keepRaw bool, hooks []llm.StreamEventHook) *stream {
	return &stream{
		body:     body,
		dec:      transport.NewSSEDecoder(body),
		provider: provider,
		keepRaw:  keepRaw,
		hooks:    hooks,
	}
}

func (s *stream) Recv() (schema.StreamEvent, error) {
	for {
		if s.done {
			return schema.StreamEvent{}, io.EOF
		}
		if len(s.pending) > 0 {
			ev := s.pending[0]
			s.pending = s.pending[1:]
			return ev, nil
		}

		data, err := s.dec.NextData()
		if err != nil {
			return schema.StreamEvent{}, err
		}

		if data == sseDoneToken {
			s.done = true
			return schema.StreamEvent{Type: schema.StreamEventDone}, nil
		}

		rawBytes := []byte(data)
		var raw json.RawMessage
		if s.keepRaw || len(s.hooks) > 0 {
			raw = json.RawMessage(rawBytes)
			if s.keepRaw {
				raw = json.RawMessage(slices.Clone(rawBytes))
			}
		}

		var chunk chatCompletionChunk
		if err := json.Unmarshal(rawBytes, &chunk); err != nil {
			return schema.StreamEvent{}, err
		}

		var mapped []schema.StreamEvent
		for _, c := range chunk.Choices {
			d := c.Delta

			if d.Content != "" || d.ReasoningContent != "" || len(d.ToolCalls) > 0 {
				ev := schema.StreamEvent{
					Type:        schema.StreamEventDelta,
					ChoiceIndex: c.Index,
					Delta:       d.Content,
					Reasoning:   d.ReasoningContent,
				}
				ev.ToolCalls = toSchemaToolCalls(d.ToolCalls)
				if s.keepRaw {
					ev.Raw = raw
				}
				mapped = append(mapped, ev)
			}

			if c.FinishReason != nil {
				fr := schema.FinishReason(*c.FinishReason)
				ev := schema.StreamEvent{
					Type:         schema.StreamEventDone,
					ChoiceIndex:  c.Index,
					FinishReason: &fr,
				}
				if s.keepRaw {
					ev.Raw = raw
				}
				if chunk.Usage != nil {
					ev.Usage = toSchemaUsagePtr(chunk.Usage)
				}
				mapped = append(mapped, ev)
			}
		}

		if len(mapped) == 0 && chunk.Usage != nil {
			mapped = append(mapped, schema.StreamEvent{
				Type:  schema.StreamEventDelta,
				Usage: toSchemaUsagePtr(chunk.Usage),
			})
			if s.keepRaw {
				mapped[len(mapped)-1].Raw = raw
			}
		}

		for _, h := range s.hooks {
			if h == nil {
				continue
			}
			for i := range mapped {
				if err := h(&mapped[i], raw); err != nil {
					return schema.StreamEvent{}, err
				}
			}
		}

		s.pending = mapped
	}
}

func (s *stream) Close() error {
	if s.done {
		return nil
	}
	s.done = true
	return s.body.Close()
}
