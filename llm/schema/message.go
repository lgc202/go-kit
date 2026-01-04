package schema

type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

// Message 聊天消息
type Message struct {
	Role Role `json:"role"`

	// Content 支持多模态内容（文本、图片、二进制等）
	Content []ContentPart `json:"content"`

	Name       string `json:"name,omitempty"`
	ToolCallID string `json:"tool_call_id,omitempty"`

	// ReasoningContent 推理内容（DeepSeek 等支持）
	ReasoningContent string     `json:"reasoning_content,omitempty"`
	ToolCalls        []ToolCall `json:"tool_calls,omitempty"`
}

// ContentPart 内容片段接口
type ContentPart interface {
	isPart()
}

// TextContent 文本内容
type TextContent struct {
	Text string
}

func (TextContent) isPart() {}

// ImageURLContent 图片 URL 内容
type ImageURLContent struct {
	URL    string
	Detail string
}

func (ImageURLContent) isPart() {}

// BinaryContent 二进制内容（如 base64 编码的图片）
type BinaryContent struct {
	MIMEType string
	Data     []byte
}

func (BinaryContent) isPart() {}

// Text 提取并拼接所有文本部分的内容
func (m Message) Text() string {
	var b []byte
	for _, p := range m.Content {
		if tp, ok := p.(TextContent); ok && tp.Text != "" {
			b = append(b, tp.Text...)
		}
	}
	return string(b)
}
