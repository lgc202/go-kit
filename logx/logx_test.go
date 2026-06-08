package logx

import (
	"bytes"
	"log/slog"
	"os"
	"strings"
	"testing"
)

func TestNewUsesJSONOutputAndDefaultFields(t *testing.T) {
	var buf bytes.Buffer
	options := Options{
		Output:  &buf,
		Format:  FormatJSON,
		Level:   LevelInfo,
		Service: "billing-api",
		Env:     "test",
		Version: "v1.2.3",
	}

	logger, err := New(options)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	logger.Info("service started")

	out := buf.String()
	for _, want := range []string{
		`"msg":"service started"`,
		`"service":"billing-api"`,
		`"env":"test"`,
		`"version":"v1.2.3"`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("log output %q does not contain %s", out, want)
		}
	}
}

func TestLoggerChangesLevelAtRuntime(t *testing.T) {
	var buf bytes.Buffer
	options := Options{
		Output: &buf,
		Format: FormatJSON,
		Level:  LevelInfo,
	}

	logger, err := New(options)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	logger.Debug("before")
	if strings.Contains(buf.String(), "before") {
		t.Fatalf("debug log was emitted before level change: %q", buf.String())
	}

	if err := logger.SetLevel(LevelDebug); err != nil {
		t.Fatalf("SetLevel() error = %v", err)
	}
	logger.Debug("after")

	if !strings.Contains(buf.String(), `"msg":"after"`) {
		t.Fatalf("debug log was not emitted after level change: %q", buf.String())
	}
	if got := logger.Level(); got != LevelDebug {
		t.Fatalf("Level() = %v, want %v", got, LevelDebug)
	}
}

func TestRedactsConfiguredSensitiveFields(t *testing.T) {
	var buf bytes.Buffer
	options := Options{
		Output:       &buf,
		Format:       FormatJSON,
		RedactFields: []string{"authorization", "token"},
	}

	logger, err := New(options)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	logger.Info("request received",
		slog.String("authorization", "Bearer secret"),
		slog.String("token", "abc123"),
		slog.String("user_id", "u-1"),
	)

	out := buf.String()
	if strings.Contains(out, "Bearer secret") || strings.Contains(out, "abc123") {
		t.Fatalf("sensitive values were not redacted: %q", out)
	}
	for _, want := range []string{
		`"authorization":"[REDACTED]"`,
		`"token":"[REDACTED]"`,
		`"user_id":"u-1"`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("log output %q does not contain %s", out, want)
		}
	}
}

func TestFileOutputWritesToConfiguredPath(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/app.log"
	options := Options{
		Format: FormatJSON,
		File: FileOptions{
			Path:       path,
			MaxSizeMB:  1,
			MaxBackups: 2,
			MaxAgeDays: 3,
		},
	}

	logger, err := New(options)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(func() {
		if err := logger.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	})

	logger.Info("file output ready")

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !strings.Contains(string(data), `"msg":"file output ready"`) {
		t.Fatalf("file log output = %q, want message", string(data))
	}
}

func TestInvalidConfigReturnsError(t *testing.T) {
	options := Options{Level: "verbose"}

	if _, err := New(options); err == nil {
		t.Fatalf("New() error = nil, want invalid level error")
	}
}

func TestFileOutputAlsoWritesToOutput(t *testing.T) {
	var buf bytes.Buffer
	dir := t.TempDir()
	path := dir + "/app.log"
	options := Options{
		Output: &buf,
		Format: FormatJSON,
		File: FileOptions{
			Path: path,
		},
	}

	logger, err := New(options)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(func() {
		if err := logger.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	})

	logger.Info("multi output ready")

	if !strings.Contains(buf.String(), `"msg":"multi output ready"`) {
		t.Fatalf("buffer log output = %q, want message", buf.String())
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !strings.Contains(string(data), `"msg":"multi output ready"`) {
		t.Fatalf("file log output = %q, want message", string(data))
	}
}
