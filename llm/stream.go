package llm

import (
	"encoding/json"
	"errors"
	"io"
)

// Stream yields StreamEvent values until io.EOF.
//
// Implementations should return io.EOF once the stream finishes normally.
type Stream interface {
	Recv() (StreamEvent, error)
	Close() error
}

type StreamEventKind string

const (
	StreamEventTextDelta      StreamEventKind = "text_delta"
	StreamEventReasoningDelta StreamEventKind = "reasoning_delta"
	StreamEventToolCallDelta  StreamEventKind = "tool_call_delta"
	StreamEventUsage          StreamEventKind = "usage"
	StreamEventDone           StreamEventKind = "done"
)

type ToolCallDelta struct {
	Index int
	ID    string
	Name  string

	ArgumentsDelta string
}

type StreamEvent struct {
	Kind StreamEventKind

	TextDelta      string
	ReasoningDelta string
	ToolCallDelta  *ToolCallDelta
	Usage          *Usage

	FinishReason FinishReason
	RawJSON      json.RawMessage
}

func (e StreamEvent) Done() bool { return e.Kind == StreamEventDone }

var ErrStreamClosed = errors.New("llm: stream closed")

// Accumulator helps build a final assistant message from a stream.
//
// It is intentionally tolerant to partial tool call deltas.
type Accumulator struct {
	Text         string
	Reasoning    string
	ToolCalls    []ToolCall
	FinishReason FinishReason
	Usage        *Usage
}

func (a *Accumulator) Apply(ev StreamEvent) {
	switch ev.Kind {
	case StreamEventTextDelta:
		a.Text += ev.TextDelta
	case StreamEventReasoningDelta:
		a.Reasoning += ev.ReasoningDelta
	case StreamEventToolCallDelta:
		if ev.ToolCallDelta == nil {
			return
		}
		idx := ev.ToolCallDelta.Index
		for len(a.ToolCalls) <= idx {
			a.ToolCalls = append(a.ToolCalls, ToolCall{})
		}
		tc := &a.ToolCalls[idx]
		if ev.ToolCallDelta.ID != "" {
			tc.ID = ev.ToolCallDelta.ID
		}
		if ev.ToolCallDelta.Name != "" {
			tc.Name = ev.ToolCallDelta.Name
		}
		tc.ArgumentsText += ev.ToolCallDelta.ArgumentsDelta
	case StreamEventUsage:
		if ev.Usage != nil {
			cpy := *ev.Usage
			a.Usage = &cpy
		}
	case StreamEventDone:
		if ev.FinishReason != "" {
			a.FinishReason = ev.FinishReason
		}
	}
}

func (a *Accumulator) FinalMessage() Message {
	msg := Message{Role: RoleAssistant, Content: a.Text, Reasoning: a.Reasoning}
	if len(a.ToolCalls) > 0 {
		msg.ToolCalls = append([]ToolCall(nil), a.ToolCalls...)
		// Best-effort: convert ArgumentsText to JSON bytes.
		for i := range msg.ToolCalls {
			if len(msg.ToolCalls[i].Arguments) == 0 && msg.ToolCalls[i].ArgumentsText != "" {
				if json.Valid([]byte(msg.ToolCalls[i].ArgumentsText)) {
					msg.ToolCalls[i].Arguments = json.RawMessage([]byte(msg.ToolCalls[i].ArgumentsText))
				}
			}
		}
	}
	return msg
}

func DrainStream(stream Stream) (ChatResponse, error) {
	defer stream.Close()

	var acc Accumulator
	for {
		ev, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return ChatResponse{}, err
		}
		acc.Apply(ev)
	}

	return ChatResponse{
		Choices: []ChatChoice{{Index: 0, Message: acc.FinalMessage(), FinishReason: acc.FinishReason}},
		Usage:   acc.Usage,
	}, nil
}
