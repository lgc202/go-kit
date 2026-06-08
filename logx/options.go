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

type options struct {
	output io.Writer
	format Format
	level  Level
	file   FileOptions

	addSource bool
	service   string
	env       string
	version   string

	redactFields []string
	redactValue  string
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

// Option 配置 Logger。
type Option interface {
	apply(*options)
}

type optionFunc func(*options)

func (f optionFunc) apply(options *options) { f(options) }

// WithOutput 设置输出目标。
func WithOutput(output io.Writer) Option {
	return optionFunc(func(options *options) {
		options.output = output
	})
}

// WithFormat 设置日志格式。
func WithFormat(format Format) Option {
	return optionFunc(func(options *options) {
		options.format = format
	})
}

// WithLevel 设置初始日志级别。
func WithLevel(level Level) Option {
	return optionFunc(func(options *options) {
		options.level = level
	})
}

// WithFile 设置文件日志和轮转参数。
func WithFile(file FileOptions) Option {
	return optionFunc(func(options *options) {
		options.file = file
	})
}

// WithAddSource 控制是否输出源码文件和行号。
func WithAddSource(enabled bool) Option {
	return optionFunc(func(options *options) {
		options.addSource = enabled
	})
}

// WithService 设置默认 service 字段。
func WithService(service string) Option {
	return optionFunc(func(options *options) {
		options.service = service
	})
}

// WithEnv 设置默认 env 字段。
func WithEnv(env string) Option {
	return optionFunc(func(options *options) {
		options.env = env
	})
}

// WithVersion 设置默认 version 字段。
func WithVersion(version string) Option {
	return optionFunc(func(options *options) {
		options.version = version
	})
}

// WithRedactFields 设置需要脱敏的字段名。
func WithRedactFields(fields ...string) Option {
	return optionFunc(func(options *options) {
		options.redactFields = append([]string(nil), fields...)
	})
}

// WithRedactDisabled 关闭字段脱敏。
func WithRedactDisabled() Option {
	return optionFunc(func(options *options) {
		options.redactFields = []string{}
	})
}

// WithRedactValue 设置脱敏后的占位值。
func WithRedactValue(value string) Option {
	return optionFunc(func(options *options) {
		options.redactValue = value
	})
}

func defaultOptions() options {
	options := options{
		format: FormatText,
		level:  LevelInfo,
	}
	if options.redactFields == nil {
		options.redactFields = defaultRedactFields()
	}
	if options.redactValue == "" {
		options.redactValue = defaultRedactValue
	}
	return options
}

func defaultAttrs(options options) []slog.Attr {
	attrs := make([]slog.Attr, 0, 3)
	if options.service != "" {
		attrs = append(attrs, slog.String("service", options.service))
	}
	if options.env != "" {
		attrs = append(attrs, slog.String("env", options.env))
	}
	if options.version != "" {
		attrs = append(attrs, slog.String("version", options.version))
	}
	return attrs
}
