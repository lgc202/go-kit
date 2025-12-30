package llm

import (
	"io"
	"testing"
)

type sliceStream struct {
	events []StreamEvent
	closed bool
}

func (s *sliceStream) Recv() (StreamEvent, error) {
	if s.closed {
		return StreamEvent{}, ErrStreamClosed
	}
	if len(s.events) == 0 {
		return StreamEvent{}, io.EOF
	}
	ev := s.events[0]
	s.events = s.events[1:]
	return ev, nil
}

func (s *sliceStream) Close() error {
	s.closed = true
	return nil
}

func TestAccumulator_MultiChoice(t *testing.T) {
	var acc Accumulator

	acc.Apply(StreamEvent{
		Kind:        StreamEventPartDelta,
		ChoiceIndex: 1,
		PartDelta:   &PartDelta{Type: ContentPartText, TextDelta: "B"},
	})
	acc.Apply(StreamEvent{
		Kind:        StreamEventPartDelta,
		ChoiceIndex: 0,
		PartDelta:   &PartDelta{Type: ContentPartText, TextDelta: "A"},
	})
	acc.Apply(StreamEvent{
		Kind:         StreamEventChoiceDone,
		ChoiceIndex:  0,
		FinishReason: FinishReasonStop,
	})
	acc.Apply(StreamEvent{
		Kind:         StreamEventChoiceDone,
		ChoiceIndex:  1,
		FinishReason: FinishReasonLength,
	})
	acc.Apply(StreamEvent{
		Kind:  StreamEventUsage,
		Usage: &Usage{PromptTokens: 1, CompletionTokens: 2, TotalTokens: 3},
	})

	resp := acc.FinalResponse()
	if got := len(resp.Choices); got != 2 {
		t.Fatalf("choices=%d", got)
	}
	if resp.Choices[0].Index != 0 || resp.Choices[0].Message.Text() != "A" {
		t.Fatalf("choice0=%+v", resp.Choices[0])
	}
	if resp.Choices[1].Index != 1 || resp.Choices[1].Message.Text() != "B" {
		t.Fatalf("choice1=%+v", resp.Choices[1])
	}
	if resp.Choices[0].FinishReason != FinishReasonStop {
		t.Fatalf("choice0.finish=%q", resp.Choices[0].FinishReason)
	}
	if resp.Choices[1].FinishReason != FinishReasonLength {
		t.Fatalf("choice1.finish=%q", resp.Choices[1].FinishReason)
	}
	if resp.Usage == nil || resp.Usage.TotalTokens != 3 {
		t.Fatalf("usage=%+v", resp.Usage)
	}
}

func TestDrainStream_BuildsResponse(t *testing.T) {
	s := &sliceStream{events: []StreamEvent{
		{Kind: StreamEventPartDelta, ChoiceIndex: 0, PartDelta: &PartDelta{Type: ContentPartText, TextDelta: "Hello"}},
		{Kind: StreamEventPartDelta, ChoiceIndex: 0, PartDelta: &PartDelta{Type: ContentPartText, TextDelta: " world"}},
		{Kind: StreamEventChoiceDone, ChoiceIndex: 0, FinishReason: FinishReasonStop},
		{Kind: StreamEventDone, ChoiceIndex: -1},
	}}

	resp, err := DrainStream(s)
	if err != nil {
		t.Fatalf("DrainStream err=%v", err)
	}
	if got := resp.FirstText(); got != "Hello world" {
		t.Fatalf("FirstText=%q", got)
	}
	if len(resp.Choices) != 1 || resp.Choices[0].FinishReason != FinishReasonStop {
		t.Fatalf("choices=%+v", resp.Choices)
	}
}
