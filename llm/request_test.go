package llm

import "testing"

func TestWithHeader_SetsTransportHeaders(t *testing.T) {
	req := BuildChatRequest("m", []Message{User("hi")}, WithHeader("X-Test", "1"))

	if req.Transport == nil || req.Transport.Headers == nil {
		t.Fatalf("expected transport headers")
	}
	if got := req.Transport.Headers.Get("X-Test"); got != "1" {
		t.Fatalf("X-Test=%q", got)
	}
}

func TestMessage_TextAndReasoning(t *testing.T) {
	msg := Message{
		Role: RoleAssistant,
		Parts: []ContentPart{
			TextPart("a"),
			ReasoningPart("r"),
			TextPart("b"),
		},
	}
	if msg.Text() != "ab" {
		t.Fatalf("Text=%q", msg.Text())
	}
	if msg.Reasoning() != "r" {
		t.Fatalf("Reasoning=%q", msg.Reasoning())
	}
}
