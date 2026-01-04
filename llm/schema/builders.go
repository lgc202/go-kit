package schema

import (
	"encoding/json"
	"fmt"
)

// TextPart 创建文本内容片段
func TextPart(text string) ContentPart {
	return TextContent{Text: text}
}

// ImageURLPart 创建图片 URL 内容片段
func ImageURLPart(url string) ContentPart {
	return ImageURLContent{URL: url}
}

// ImageURLWithDetailPart 创建指定 detail 的图片内容片段
func ImageURLWithDetailPart(url, detail string) ContentPart {
	return ImageURLContent{URL: url, Detail: detail}
}

// BinaryPart 创建二进制内容片段
func BinaryPart(mimeType string, data []byte) ContentPart {
	return BinaryContent{MIMEType: mimeType, Data: data}
}

// SystemMessage 创建系统消息
func SystemMessage(content string) Message {
	return Message{Role: RoleSystem, Content: []ContentPart{TextPart(content)}}
}

// UserMessage 创建用户消息
func UserMessage(content string) Message {
	return Message{Role: RoleUser, Content: []ContentPart{TextPart(content)}}
}

// AssistantMessage 创建助手消息
func AssistantMessage(content string) Message {
	return Message{Role: RoleAssistant, Content: []ContentPart{TextPart(content)}}
}

// ToolResultMessage 创建工具调用结果消息
func ToolResultMessage(toolCallID, content string) Message {
	return Message{Role: RoleTool, ToolCallID: toolCallID, Content: []ContentPart{TextPart(content)}}
}

// JSON 将任意类型转换为 JSON RawMessage
func JSON(v any) (json.RawMessage, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(b), nil
}

// MustJSON 将任意类型转换为 JSON RawMessage，失败时 panic
func MustJSON(v any) json.RawMessage {
	b, err := JSON(v)
	if err != nil {
		panic(err)
	}
	return b
}

// NewFunctionTool 创建函数调用工具
func NewFunctionTool(name, description string, parameters any) (Tool, error) {
	if name == "" {
		return Tool{}, fmt.Errorf("function name required")
	}

	fd := FunctionDefinition{
		Name:        name,
		Description: description,
	}
	if parameters != nil {
		b, err := json.Marshal(parameters)
		if err != nil {
			return Tool{}, fmt.Errorf("marshal parameters: %w", err)
		}
		fd.Parameters = json.RawMessage(b)
	}

	return Tool{
		Type:     ToolTypeFunction,
		Function: fd,
	}, nil
}
