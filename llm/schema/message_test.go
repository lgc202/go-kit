package schema

import (
	"testing"
)

// TestMessageText 测试 Message.Text() 方法 - 核心逻辑
func TestMessageText(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		msg  Message
		want string
	}{
		{
			name: "single text part",
			msg: Message{
				Content: []ContentPart{TextContent{Text: "Hello"}},
			},
			want: "Hello",
		},
		{
			name: "multiple text parts - should concatenate",
			msg: Message{
				Content: []ContentPart{
					TextContent{Text: "Hello"},
					TextContent{Text: " World"},
				},
			},
			want: "Hello World",
		},
		{
			name: "mixed content - should only extract text",
			msg: Message{
				Content: []ContentPart{
					TextContent{Text: "Hello"},
					ImageURLContent{URL: "http://example.com/image.png"},
					TextContent{Text: " World"},
				},
			},
			want: "Hello World",
		},
		{
			name: "empty content",
			msg:  Message{},
			want: "",
		},
		{
			name: "only image content - no text",
			msg: Message{
				Content: []ContentPart{
					ImageURLContent{URL: "http://example.com/image.png"},
				},
			},
			want: "",
		},
		{
			name: "empty text parts - should skip",
			msg: Message{
				Content: []ContentPart{
					TextContent{Text: ""},
					TextContent{Text: "Hello"},
					TextContent{Text: ""},
				},
			},
			want: "Hello",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.msg.Text(); got != tt.want {
				t.Errorf("Message.Text() = %q, want %q", got, tt.want)
			}
		})
	}
}
