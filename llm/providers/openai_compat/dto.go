package openai_compat

import (
	"encoding/json"
	"strconv"
	"time"
)

// apiMessage / api* types model OpenAI-compatible "wire" payloads.
// They are intentionally distinct from llm domain types.
type apiMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
	Name    string `json:"name,omitempty"`

	ReasoningContent string `json:"reasoning_content,omitempty"`
	Thinking         any    `json:"thinking,omitempty"`

	ToolCallID string        `json:"tool_call_id,omitempty"`
	ToolCalls  []apiToolCall `json:"tool_calls,omitempty"`
}

type apiToolCall struct {
	Index    int             `json:"index,omitempty"`
	ID       string          `json:"id,omitempty"`
	Type     string          `json:"type,omitempty"`
	Function apiFunctionCall `json:"function,omitempty"`
}

type apiFunctionCall struct {
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

type apiTool struct {
	Type     string         `json:"type"`
	Function apiFunctionDef `json:"function"`
}

type apiFunctionDef struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
}

type chatCompletionResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`

	Choices []chatCompletionChoice `json:"choices"`
	Usage   *chatCompletionUsage   `json:"usage,omitempty"`
}

func (r chatCompletionResponse) CreatedTime() time.Time {
	if r.Created <= 0 {
		return time.Time{}
	}
	return time.Unix(r.Created, 0).UTC()
}

type chatCompletionChoice struct {
	Index        int        `json:"index"`
	Message      apiMessage `json:"message"`
	FinishReason string     `json:"finish_reason"`
}

type chatCompletionUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`

	raw map[string]json.RawMessage
}

func (u *chatCompletionUsage) UnmarshalJSON(b []byte) error {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(b, &m); err != nil {
		return err
	}
	u.raw = m

	_ = json.Unmarshal(m["prompt_tokens"], &u.PromptTokens)
	_ = json.Unmarshal(m["completion_tokens"], &u.CompletionTokens)
	_ = json.Unmarshal(m["total_tokens"], &u.TotalTokens)
	return nil
}

func (u *chatCompletionUsage) intField(key string) int {
	if u == nil || u.raw == nil {
		return 0
	}
	b, ok := u.raw[key]
	if !ok || len(b) == 0 {
		return 0
	}
	var n int
	if err := json.Unmarshal(b, &n); err == nil {
		return n
	}
	// Some providers might encode numbers as strings.
	var s string
	if err := json.Unmarshal(b, &s); err == nil {
		if v, err := strconv.Atoi(s); err == nil {
			return v
		}
	}
	return 0
}

func (u *chatCompletionUsage) intFieldInObject(objKey, key string) int {
	if u == nil || u.raw == nil {
		return 0
	}
	rawObj, ok := u.raw[objKey]
	if !ok || len(rawObj) == 0 {
		return 0
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(rawObj, &m); err != nil {
		return 0
	}
	rawVal, ok := m[key]
	if !ok || len(rawVal) == 0 {
		return 0
	}
	var n int
	if err := json.Unmarshal(rawVal, &n); err == nil {
		return n
	}
	var s string
	if err := json.Unmarshal(rawVal, &s); err == nil {
		if v, err := strconv.Atoi(s); err == nil {
			return v
		}
	}
	return 0
}

type errorEnvelope struct {
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    any    `json:"code"`
	} `json:"error"`
}

type chatCompletionChunk struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`

	Choices []chatCompletionChunkChoice `json:"choices"`
	Usage   *chatCompletionUsage        `json:"usage,omitempty"`
	Error   *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    any    `json:"code"`
	} `json:"error,omitempty"`
}

type chatCompletionChunkChoice struct {
	Index        int                 `json:"index"`
	Delta        chatCompletionDelta `json:"delta"`
	FinishReason string              `json:"finish_reason"`
}

type chatCompletionDelta struct {
	Role             string        `json:"role,omitempty"`
	Content          any           `json:"content,omitempty"`
	ReasoningContent string        `json:"reasoning_content,omitempty"`
	Thinking         any           `json:"thinking,omitempty"`
	ToolCalls        []apiToolCall `json:"tool_calls,omitempty"`
}
