package version

import (
	"encoding/json"
	"runtime"
	"strings"
	"testing"
)

func TestInfo_String(t *testing.T) {
	tests := []struct {
		name     string
		info     Info
		expected string
	}{
		{
			name: "clean state",
			info: Info{
				GitVersion:   "v1.0.0",
				GitTreeState: "clean",
			},
			expected: "v1.0.0",
		},
		{
			name: "dirty state",
			info: Info{
				GitVersion:   "v1.0.0",
				GitTreeState: "dirty",
			},
			expected: "v1.0.0-dirty",
		},
		{
			name: "empty state",
			info: Info{
				GitVersion:   "v1.0.0",
				GitTreeState: "",
			},
			expected: "v1.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.info.String(); got != tt.expected {
				t.Errorf("Info.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestInfo_ShortString(t *testing.T) {
	info := Info{
		GitVersion:   "v1.0.0",
		GitTreeState: "dirty",
	}

	if got := info.ShortString(); got != "v1.0.0" {
		t.Errorf("ShortString() = %v, want v1.0.0", got)
	}
}

func TestInfo_ToJSON(t *testing.T) {
	info := Info{
		GitVersion:   "v1.0.0",
		GitCommit:    "abc123",
		GitTreeState: "clean",
		BuildDate:    "2024-01-01T00:00:00Z",
		GoVersion:    "go1.21.0",
		Compiler:     "gc",
		Platform:     "linux/amd64",
	}

	jsonStr, err := info.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() error = %v", err)
	}

	var parsed Info
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if parsed.GitVersion != info.GitVersion {
		t.Errorf("GitVersion = %v, want %v", parsed.GitVersion, info.GitVersion)
	}
}

func TestInfo_ToJSONIndent(t *testing.T) {
	info := Info{
		GitVersion: "v1.0.0",
		GitCommit:  "abc123",
	}

	jsonStr, err := info.ToJSONIndent()
	if err != nil {
		t.Fatalf("ToJSONIndent() error = %v", err)
	}

	// 验证是格式化的 JSON（包含换行）
	if !strings.Contains(jsonStr, "\n") {
		t.Error("ToJSONIndent() should return formatted JSON with newlines")
	}
}

func TestInfo_Text(t *testing.T) {
	info := Info{
		GitVersion:   "v1.0.0",
		GitCommit:    "abc123",
		GitTreeState: "clean",
		BuildDate:    "2024-01-01T00:00:00Z",
		GoVersion:    "go1.21.0",
		Compiler:     "gc",
		Platform:     "linux/amd64",
	}

	text := info.Text()

	// 验证所有字段都在输出中
	expectedFields := []string{
		"gitVersion:", "v1.0.0",
		"gitCommit:", "abc123",
		"gitTreeState:", "clean",
		"buildDate:", "2024-01-01T00:00:00Z",
		"goVersion:", "go1.21.0",
		"compiler:", "gc",
		"platform:", "linux/amd64",
	}

	for _, field := range expectedFields {
		if !strings.Contains(text, field) {
			t.Errorf("Text() missing field %q", field)
		}
	}
}

func TestInfo_Text_OmitEmpty(t *testing.T) {
	info := Info{
		GitVersion: "v1.0.0",
		GitCommit:  "abc123",
		BuildDate:  "2024-01-01T00:00:00Z",
		GoVersion:  "go1.21.0",
		Compiler:   "gc",
		Platform:   "linux/amd64",
	}

	text := info.Text()

	// 空字段不应该出现
	if strings.Contains(text, "gitTreeState:") {
		t.Error("Text() should not contain empty gitTreeState")
	}
}

func TestGet(t *testing.T) {
	info := Get()

	if info.GoVersion != runtime.Version() {
		t.Errorf("GoVersion = %v, want %v", info.GoVersion, runtime.Version())
	}

	if info.Compiler != runtime.Compiler {
		t.Errorf("Compiler = %v, want %v", info.Compiler, runtime.Compiler)
	}

	// 验证 Platform 格式
	if !strings.Contains(info.Platform, "/") {
		t.Errorf("Platform should contain '/', got %v", info.Platform)
	}
}
