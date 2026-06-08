package logx

import (
	"io"
	"log/slog"
)

const (
	defaultFileMaxSizeMB  = 100
	defaultFileMaxBackups = 10
	defaultFileMaxAgeDays = 30
	defaultRedactValue    = "[REDACTED]"
)

// Format 表示日志输出格式。
type Format string

const (
	// FormatText 表示适合本地阅读的文本日志。
	FormatText Format = "text"
	// FormatJSON 表示适合生产采集的 JSON 日志。
	FormatJSON Format = "json"
)

// Level 表示日志级别。
type Level string

const (
	// LevelDebug 表示调试日志。
	LevelDebug Level = "debug"
	// LevelInfo 表示普通运行日志。
	LevelInfo Level = "info"
	// LevelWarn 表示可恢复异常日志。
	LevelWarn Level = "warn"
	// LevelError 表示需要关注的错误日志。
	LevelError Level = "error"
)

// Options 定义 logger 初始化参数。
type Options struct {
	Output io.Writer
	Format Format
	Level  Level
	File   FileOptions

	AddSource bool
	Service   string
	Env       string
	Version   string

	RedactFields []string
	RedactValue  string
}

// FileOptions 定义文件日志和轮转参数。
type FileOptions struct {
	Path       string
	MaxSizeMB  int
	MaxBackups int
	MaxAgeDays int
	Compress   bool
	LocalTime  bool
}

func defaultOptions(options Options) Options {
	if options.Format == "" {
		options.Format = FormatText
	}
	if options.Level == "" {
		options.Level = LevelInfo
	}
	if options.RedactFields == nil {
		options.RedactFields = defaultRedactFields()
	}
	if options.RedactValue == "" {
		options.RedactValue = defaultRedactValue
	}
	return options
}

func defaultAttrs(options Options) []slog.Attr {
	attrs := make([]slog.Attr, 0, 3)
	if options.Service != "" {
		attrs = append(attrs, slog.String("service", options.Service))
	}
	if options.Env != "" {
		attrs = append(attrs, slog.String("env", options.Env))
	}
	if options.Version != "" {
		attrs = append(attrs, slog.String("version", options.Version))
	}
	return attrs
}
