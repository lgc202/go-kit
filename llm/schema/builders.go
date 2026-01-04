package schema

import (
	"encoding/json"
	"fmt"
)

func TextPart(text string) ContentPart {
	return TextContent{Text: text}
}

func ImageURLPart(url string) ContentPart {
	return ImageURLContent{URL: url}
}

func ImageURLWithDetailPart(url, detail string) ContentPart {
	return ImageURLContent{URL: url, Detail: detail}
}

func BinaryPart(mimeType string, data []byte) ContentPart {
	return BinaryContent{MIMEType: mimeType, Data: data}
}

func SystemMessage(content string) Message {
	return Message{Role: RoleSystem, Content: []ContentPart{TextPart(content)}}
}

func UserMessage(content string) Message {
	return Message{Role: RoleUser, Content: []ContentPart{TextPart(content)}}
}

func AssistantMessage(content string) Message {
	return Message{Role: RoleAssistant, Content: []ContentPart{TextPart(content)}}
}

func ToolResultMessage(toolCallID, content string) Message {
	return Message{Role: RoleTool, ToolCallID: toolCallID, Content: []ContentPart{TextPart(content)}}
}

func JSON(v any) (json.RawMessage, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(b), nil
}

func MustJSON(v any) json.RawMessage {
	b, err := JSON(v)
	if err != nil {
		panic(err)
	}
	return b
}

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
