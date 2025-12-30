package llm

import (
	"encoding/json"
	"errors"
	"io"
	"sort"
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
	StreamEventPartDelta     StreamEventKind = "part_delta"
	StreamEventToolCallDelta StreamEventKind = "tool_call_delta"
	StreamEventUsage         StreamEventKind = "usage"
	StreamEventChoiceDone    StreamEventKind = "choice_done"
	StreamEventDone          StreamEventKind = "done"
)

type ToolCallDelta struct {
	Index int
	ID    string
	Name  string

	ArgumentsDelta string
}

type PartDelta struct {
	Type ContentPartType

	// TextDelta appends to the corresponding part's text content.
	TextDelta string
}

type StreamEvent struct {
	Kind StreamEventKind

	// ChoiceIndex is meaningful for choice-scoped events (part/tool_call/choice_done).
	// For usage/done events it may be -1.
	ChoiceIndex int

	PartDelta     *PartDelta
	ToolCallDelta *ToolCallDelta
	Usage         *Usage

	FinishReason FinishReason
	RawJSON      json.RawMessage
}

func (e StreamEvent) Done() bool { return e.Kind == StreamEventDone }

var ErrStreamClosed = errors.New("llm: stream closed")

type ChoiceAccumulator struct {
	Parts        []ContentPart
	ToolCalls    []ToolCall
	FinishReason FinishReason
}

func (a *ChoiceAccumulator) appendPartDelta(d PartDelta) {
	if d.TextDelta == "" {
		return
	}
	if n := len(a.Parts); n > 0 && a.Parts[n-1].Type == d.Type {
		a.Parts[n-1].Text += d.TextDelta
		return
	}
	a.Parts = append(a.Parts, ContentPart{Type: d.Type, Text: d.TextDelta})
}

func (a *ChoiceAccumulator) applyToolCallDelta(d ToolCallDelta) {
	idx := d.Index
	for len(a.ToolCalls) <= idx {
		a.ToolCalls = append(a.ToolCalls, ToolCall{})
	}
	tc := &a.ToolCalls[idx]
	if d.ID != "" {
		tc.ID = d.ID
	}
	if d.Name != "" {
		tc.Name = d.Name
	}
	tc.ArgumentsText += d.ArgumentsDelta
}

func (a *ChoiceAccumulator) FinalMessage() Message {
	msg := Message{Role: RoleAssistant}
	if len(a.Parts) > 0 {
		msg.Parts = make([]ContentPart, len(a.Parts))
		copy(msg.Parts, a.Parts)
		for i := range msg.Parts {
			msg.Parts[i].JSON = append([]byte(nil), msg.Parts[i].JSON...)
			msg.Parts[i].Data = append([]byte(nil), msg.Parts[i].Data...)
		}
	}
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

// Accumulator builds a final response from streaming events.
//
// It supports multiple choices (ChoiceIndex) and is intentionally tolerant to
// partial tool call deltas.
type Accumulator struct {
	choices map[int]*ChoiceAccumulator
	Usage   *Usage
}

func (a *Accumulator) choice(idx int) *ChoiceAccumulator {
	if a.choices == nil {
		a.choices = make(map[int]*ChoiceAccumulator)
	}
	if a.choices[idx] == nil {
		a.choices[idx] = &ChoiceAccumulator{}
	}
	return a.choices[idx]
}

func (a *Accumulator) Apply(ev StreamEvent) {
	switch ev.Kind {
	case StreamEventPartDelta:
		if ev.PartDelta == nil {
			return
		}
		a.choice(ev.ChoiceIndex).appendPartDelta(*ev.PartDelta)
	case StreamEventToolCallDelta:
		if ev.ToolCallDelta == nil {
			return
		}
		a.choice(ev.ChoiceIndex).applyToolCallDelta(*ev.ToolCallDelta)
	case StreamEventChoiceDone:
		if ev.FinishReason != "" {
			a.choice(ev.ChoiceIndex).FinishReason = ev.FinishReason
		}
	case StreamEventUsage:
		if ev.Usage != nil {
			cpy := *ev.Usage
			a.Usage = &cpy
		}
	case StreamEventDone:
		return
	}
}

func (a *Accumulator) FinalResponse() ChatResponse {
	if len(a.choices) == 0 {
		return ChatResponse{Usage: a.Usage}
	}

	idxs := make([]int, 0, len(a.choices))
	for idx := range a.choices {
		idxs = append(idxs, idx)
	}
	sort.Ints(idxs)

	out := ChatResponse{Choices: make([]ChatChoice, 0, len(idxs)), Usage: a.Usage}
	for _, idx := range idxs {
		ca := a.choices[idx]
		out.Choices = append(out.Choices, ChatChoice{
			Index:        idx,
			Message:      ca.FinalMessage(),
			FinishReason: ca.FinishReason,
		})
	}
	return out
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

	return acc.FinalResponse(), nil
}
