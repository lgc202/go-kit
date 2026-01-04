package schema

import (
	"testing"
)

// TestNewFunctionTool 测试工具创建 - 包含验证逻辑
func TestNewFunctionTool(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		nameParam   string
		description string
		parameters  any
		wantErr     bool
		errContains string
	}{
		{
			name:        "valid tool with parameters",
			nameParam:   "get_weather",
			description: "Get weather",
			parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"location": map[string]any{"type": "string"},
				},
			},
			wantErr: false,
		},
		{
			name:        "valid tool without parameters",
			nameParam:   "simple_tool",
			description: "A simple tool",
			parameters:  nil,
			wantErr:     false,
		},
		{
			name:        "empty name - error",
			nameParam:   "",
			description: "Tool",
			parameters:  nil,
			wantErr:     true,
			errContains: "function name required",
		},
		{
			name:        "invalid parameters - error",
			nameParam:   "invalid_tool",
			description: "Tool",
			parameters:  make(chan int),
			wantErr:     true,
			errContains: "marshal parameters",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewFunctionTool(tt.nameParam, tt.description, tt.parameters)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewFunctionTool() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				if tt.errContains != "" && err != nil {
					if !contains(err.Error(), tt.errContains) {
						t.Errorf("error = %q, want contain %q", err.Error(), tt.errContains)
					}
				}
				return
			}
			if got.Type != ToolTypeFunction {
				t.Errorf("Type = %v, want %v", got.Type, ToolTypeFunction)
			}
			if got.Function.Name != tt.nameParam {
				t.Errorf("Function.Name = %v, want %v", got.Function.Name, tt.nameParam)
			}
		})
	}
}

// TestJSON 测试 JSON 序列化 - 包含错误处理逻辑
func TestJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		v       any
		want    string
		wantErr bool
	}{
		{
			name:    "object",
			v:       map[string]any{"key": "value"},
			want:    `{"key":"value"}`,
			wantErr: false,
		},
		{
			name:    "array",
			v:       []any{"a", "b"},
			want:    `["a","b"]`,
			wantErr: false,
		},
		{
			name:    "nil",
			v:       nil,
			want:    `null`,
			wantErr: false,
		},
		{
			name:    "invalid - channel",
			v:       make(chan int),
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := JSON(tt.v)
			if (err != nil) != tt.wantErr {
				t.Errorf("JSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && string(got) != tt.want {
				t.Errorf("JSON() = %q, want %q", string(got), tt.want)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
