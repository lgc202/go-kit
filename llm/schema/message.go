package schema

type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

type Message struct {
	Role Role `json:"role"`

	// Content 包含结构化内容（如文本 + 图片）
	//
	// 对于简单文本消息，使用单个 TextContent 部分（通过 schema.TextPart）
	Content []ContentPart `json:"content"`

	// 可选字段，并非所有 provider 都支持/接受这些字段
	Name       string `json:"name,omitempty"`
	ToolCallID string `json:"tool_call_id,omitempty"`

	// 可选字段，用于返回独立推理内容和工具调用的 provider（如 DeepSeek）
	ReasoningContent string     `json:"reasoning_content,omitempty"`
	ToolCalls        []ToolCall `json:"tool_calls,omitempty"`
}

type ContentPart interface {
	isPart()
}

type TextContent struct {
	Text string
}

func (TextContent) isPart() {}

type ImageURLContent struct {
	URL    string
	Detail string
}

func (ImageURLContent) isPart() {}

type BinaryContent struct {
	MIMEType string
	Data     []byte
}

func (BinaryContent) isPart() {}

// Text returns the concatenated plain text of all text parts.
func (m Message) Text() string {
	var b []byte
	for _, p := range m.Content {
		if tp, ok := p.(TextContent); ok && tp.Text != "" {
			b = append(b, tp.Text...)
		}
	}
	return string(b)
}
