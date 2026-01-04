package schema

import "encoding/json"

// ToolCallType 表示工具调用的类型
type ToolCallType string

const (
	ToolCallTypeFunction ToolCallType = "function"
)

// ToolCall 表示模型发起的工具调用
type ToolCall struct {
	ID       string       `json:"id"`
	Type     ToolCallType `json:"type"`
	Function ToolFunction `json:"function"`
}

// ToolFunction 表示要调用的函数及其参数
type ToolFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// ToolType 表示工具的类型
type ToolType string

const (
	ToolTypeFunction ToolType = "function"
)

// Tool 表示可供给模型使用的工具
type Tool struct {
	Type     ToolType           `json:"type"`
	Function FunctionDefinition `json:"function"`
}

// FunctionDefinition 定义一个函数的名称、描述和参数 schema
type FunctionDefinition struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`

	// Strict 是否严格按照 schema 进行结构化输出，依赖于 provider 支持
	Strict bool `json:"strict,omitempty"`
}

// ToolChoiceMode 表示工具选择模式
type ToolChoiceMode string

const (
	ToolChoiceNone ToolChoiceMode = "none" // 不调用工具
	ToolChoiceAuto ToolChoiceMode = "auto" // 自动决定是否调用工具
)

// ToolChoice 控制模型何时以及如何调用工具
type ToolChoice struct {
	Mode         ToolChoiceMode `json:"mode"`
	FunctionName string         `json:"function_name,omitempty"`
}

// ResponseFormat 控制模型返回的格式
type ResponseFormat struct {
	Type string `json:"type"`

	// JSONSchema 当 Type 为 "json_schema" 时使用的 schema 定义，依赖于 provider 支持
	JSONSchema json.RawMessage `json:"json_schema,omitempty"`
}
