package openai_compat

import (
	"encoding/json"
	"io"

	"github.com/lgc202/go-kit/llm/schema"
)

type stream struct {
	body io.ReadCloser
	dec  *sseDecoder

	adapter  Adapter
	provider string
	keepRaw  bool

	pending []schema.StreamEvent
	done    bool
}

func newStream(provider string, adapter Adapter, body io.ReadCloser, keepRaw bool) *stream {
	return &stream{
		body:     body,
		dec:      newSSEDecoder(body),
		adapter:  adapter,
		provider: provider,
		keepRaw:  keepRaw,
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

		if data == "[DONE]" {
			s.done = true
			return schema.StreamEvent{Type: schema.StreamEventDone}, nil
		}

		var chunk chatCompletionChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
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
				if len(d.ToolCalls) > 0 {
					ev.ToolCalls = make([]schema.ToolCall, 0, len(d.ToolCalls))
					for _, tc := range d.ToolCalls {
						ev.ToolCalls = append(ev.ToolCalls, schema.ToolCall{
							ID:   tc.ID,
							Type: schema.ToolCallType(tc.Type),
							Function: schema.ToolFunction{
								Name:      tc.Function.Name,
								Arguments: tc.Function.Arguments,
							},
						})
					}
				}
				if s.adapter != nil {
					if err := s.adapter.EnrichStreamDelta(&ev, nil); err != nil {
						return schema.StreamEvent{}, err
					}
				}
				if s.keepRaw {
					ev.Raw = json.RawMessage([]byte(data))
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
					ev.Raw = json.RawMessage([]byte(data))
				}
				if chunk.Usage != nil {
					ev.Usage = &schema.Usage{
						PromptTokens:          chunk.Usage.PromptTokens,
						CompletionTokens:      chunk.Usage.CompletionTokens,
						TotalTokens:           chunk.Usage.TotalTokens,
						PromptCacheHitTokens:  chunk.Usage.PromptCacheHitTokens,
						PromptCacheMissTokens: chunk.Usage.PromptCacheMissTokens,
					}
					if chunk.Usage.CompletionTokensDetails != nil && chunk.Usage.CompletionTokensDetails.ReasoningTokens != 0 {
						ev.Usage.CompletionTokensDetails = &schema.CompletionTokensDetails{
							ReasoningTokens: chunk.Usage.CompletionTokensDetails.ReasoningTokens,
						}
					}
				}
				mapped = append(mapped, ev)
			}
		}

		if len(mapped) == 0 && chunk.Usage != nil {
			mapped = append(mapped, schema.StreamEvent{
				Type: schema.StreamEventDelta,
				Usage: &schema.Usage{
					PromptTokens:          chunk.Usage.PromptTokens,
					CompletionTokens:      chunk.Usage.CompletionTokens,
					TotalTokens:           chunk.Usage.TotalTokens,
					PromptCacheHitTokens:  chunk.Usage.PromptCacheHitTokens,
					PromptCacheMissTokens: chunk.Usage.PromptCacheMissTokens,
				},
			})
			if u := mapped[len(mapped)-1].Usage; u != nil && chunk.Usage.CompletionTokensDetails != nil && chunk.Usage.CompletionTokensDetails.ReasoningTokens != 0 {
				u.CompletionTokensDetails = &schema.CompletionTokensDetails{
					ReasoningTokens: chunk.Usage.CompletionTokensDetails.ReasoningTokens,
				}
			}
		}

		// 将映射的事件添加到待处理列表
		// 如果 mapped 为空，循环继续读取下一个数据块
		s.pending = mapped
		// 循环继续，会检查 pending 列表并返回
	}
}

func (s *stream) Close() error {
	if s.done {
		return nil
	}
	s.done = true
	return s.body.Close()
}
