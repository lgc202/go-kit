package llm

import (
	"context"
	"errors"
	"io"
	"slices"
	"strings"

	"github.com/lgc202/go-kit/llm/schema"
)

type ClientOption func(*Client)

type Client struct {
	model       ChatModel
	defaultOpts []RequestOption
}

var _ ChatModel = (*Client)(nil)
var _ ProviderNamer = (*Client)(nil)

func Wrap(model ChatModel, opts ...ClientOption) *Client {
	c := &Client{model: model}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		opt(c)
	}
	return c
}

func WithDefaultRequestOptions(opts ...RequestOption) ClientOption {
	return func(c *Client) {
		c.defaultOpts = append(c.defaultOpts, opts...)
	}
}

func (c *Client) Chat(ctx context.Context, messages []schema.Message, opts ...RequestOption) (schema.ChatResponse, error) {
	merged := slices.Concat(c.defaultOpts, opts)
	reqCfg := ApplyRequestOptions(merged...)
	if reqCfg.StreamingFunc == nil && reqCfg.StreamingReasoningFunc == nil {
		return c.model.Chat(ctx, messages, merged...)
	}

	s, err := c.model.ChatStream(ctx, messages, merged...)
	if err != nil {
		return schema.ChatResponse{}, err
	}
	defer s.Close()

	type choiceAgg struct {
		content   strings.Builder
		reasoning strings.Builder

		toolCalls     []schema.ToolCall
		toolCallIndex map[string]int

		finishReason schema.FinishReason
		hasFinish    bool
	}

	choices := make(map[int]*choiceAgg)
	getChoice := func(idx int) *choiceAgg {
		if a, ok := choices[idx]; ok {
			return a
		}
		a := &choiceAgg{
			toolCallIndex: make(map[string]int),
		}
		choices[idx] = a
		return a
	}

	var usage *schema.Usage

	for {
		ev, err := s.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return schema.ChatResponse{}, err
		}

		switch ev.Type {
		case schema.StreamEventDelta:
			if ev.Usage != nil {
				u := *ev.Usage
				usage = &u
			}

			if ev.Delta == "" && ev.Reasoning == "" && len(ev.ToolCalls) == 0 {
				continue
			}

			a := getChoice(ev.ChoiceIndex)
			if ev.Delta != "" {
				a.content.WriteString(ev.Delta)
			}
			if ev.Reasoning != "" {
				a.reasoning.WriteString(ev.Reasoning)
			}

			if len(ev.ToolCalls) > 0 {
				for _, tc := range ev.ToolCalls {
					pos, ok := a.toolCallIndex[tc.ID]
					if !ok {
						a.toolCallIndex[tc.ID] = len(a.toolCalls)
						a.toolCalls = append(a.toolCalls, tc)
						continue
					}

					existing := &a.toolCalls[pos]
					if existing.Type == "" {
						existing.Type = tc.Type
					}
					if existing.Function.Name == "" {
						existing.Function.Name = tc.Function.Name
					}
					if tc.Function.Arguments != "" {
						switch {
						case len(tc.Function.Arguments) > len(existing.Function.Arguments) && strings.HasPrefix(tc.Function.Arguments, existing.Function.Arguments):
							existing.Function.Arguments = tc.Function.Arguments
						case len(existing.Function.Arguments) > len(tc.Function.Arguments) && strings.HasPrefix(existing.Function.Arguments, tc.Function.Arguments):
						default:
							existing.Function.Arguments += tc.Function.Arguments
						}
					}
				}
			}

			if reqCfg.StreamingReasoningFunc != nil {
				if err := reqCfg.StreamingReasoningFunc(ctx, []byte(ev.Reasoning), []byte(ev.Delta)); err != nil {
					return schema.ChatResponse{}, err
				}
			} else if reqCfg.StreamingFunc != nil && ev.Delta != "" {
				if err := reqCfg.StreamingFunc(ctx, []byte(ev.Delta)); err != nil {
					return schema.ChatResponse{}, err
				}
			}

		case schema.StreamEventDone:
			if ev.Usage != nil {
				u := *ev.Usage
				usage = &u
			}

			if ev.FinishReason == nil {
				goto done
			}

			a := getChoice(ev.ChoiceIndex)
			a.finishReason = *ev.FinishReason
			a.hasFinish = true
		}
	}

done:
	resp := schema.ChatResponse{
		Model: reqCfg.Model,
	}
	if usage != nil {
		resp.Usage = *usage
	}

	maxIdx := -1
	for idx := range choices {
		if idx > maxIdx {
			maxIdx = idx
		}
	}
	if maxIdx >= 0 {
		resp.Choices = make([]schema.Choice, 0, maxIdx+1)
		for i := 0; i <= maxIdx; i++ {
			a, ok := choices[i]
			if !ok {
				continue
			}
			fr := a.finishReason
			if !a.hasFinish {
				fr = ""
			}

			msg := schema.Message{
				Role:             schema.RoleAssistant,
				Content:          []schema.ContentPart{schema.TextPart(a.content.String())},
				ReasoningContent: a.reasoning.String(),
			}
			if len(a.toolCalls) > 0 {
				msg.ToolCalls = slices.Clone(a.toolCalls)
			}

			resp.Choices = append(resp.Choices, schema.Choice{
				Index:        i,
				Message:      msg,
				FinishReason: fr,
			})
		}
	}

	return resp, nil
}

func (c *Client) ChatStream(ctx context.Context, messages []schema.Message, opts ...RequestOption) (Stream, error) {
	merged := slices.Concat(c.defaultOpts, opts)
	return c.model.ChatStream(ctx, messages, merged...)
}

func (c *Client) Provider() Provider {
	if p, ok := c.model.(ProviderNamer); ok {
		return p.Provider()
	}

	return ProviderUnknown
}
