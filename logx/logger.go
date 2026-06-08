package logx

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	lumberjack "gopkg.in/natefinch/lumberjack.v2"
)

// Logger 是带运行时控制能力的 slog.Logger。
type Logger struct {
	*slog.Logger

	level  slog.LevelVar
	closer io.Closer
}

// Level 返回当前日志级别。
func (l *Logger) Level() Level {
	if l == nil {
		return LevelInfo
	}
	switch l.level.Level() {
	case slog.LevelDebug:
		return LevelDebug
	case slog.LevelInfo:
		return LevelInfo
	case slog.LevelWarn:
		return LevelWarn
	case slog.LevelError:
		return LevelError
	default:
		return Level(l.level.Level().String())
	}
}

// SetLevel 修改日志级别。
func (l *Logger) SetLevel(level Level) error {
	if l == nil {
		return nil
	}
	parsed, err := parseLevel(level)
	if err != nil {
		return err
	}
	l.level.Set(parsed)
	return nil
}

// Close 关闭底层输出。文件输出需要调用 Close。
func (l *Logger) Close() error {
	if l == nil || l.closer == nil {
		return nil
	}
	return l.closer.Close()
}

// New 根据 options 创建 Logger。
func New(options Options) (*Logger, error) {
	options = defaultOptions(options)

	level, err := parseLevel(options.Level)
	if err != nil {
		return nil, err
	}

	w, closer := buildOutput(options.Output, options.File)
	if w == nil {
		w = os.Stdout
	}

	logger := &Logger{closer: closer}
	logger.level.Set(level)

	opts := &slog.HandlerOptions{
		AddSource:   options.AddSource,
		Level:       &logger.level,
		ReplaceAttr: redactAttr(options),
	}

	var handler slog.Handler
	switch options.Format {
	case FormatText:
		handler = slog.NewTextHandler(w, opts)
	case FormatJSON:
		handler = slog.NewJSONHandler(w, opts)
	default:
		return nil, fmt.Errorf("logx: unsupported format %q", options.Format)
	}

	logger.Logger = slog.New(handler)
	if attrs := defaultAttrs(options); len(attrs) > 0 {
		args := make([]any, 0, len(attrs)*2)
		for _, attr := range attrs {
			args = append(args, attr.Key, attr.Value.Any())
		}
		logger.Logger = logger.With(args...)
	}

	return logger, nil
}

func parseLevel(level Level) (slog.Level, error) {
	switch Level(strings.ToLower(strings.TrimSpace(string(level)))) {
	case "", LevelInfo:
		return slog.LevelInfo, nil
	case LevelDebug:
		return slog.LevelDebug, nil
	case LevelWarn, "warning":
		return slog.LevelWarn, nil
	case LevelError:
		return slog.LevelError, nil
	default:
		return slog.LevelInfo, fmt.Errorf("logx: unsupported level %q", level)
	}
}

func buildOutput(output io.Writer, file FileOptions) (io.Writer, io.Closer) {
	if file.Path == "" {
		return output, nil
	}

	if file.MaxSizeMB == 0 {
		file.MaxSizeMB = defaultFileMaxSizeMB
	}
	if file.MaxBackups == 0 {
		file.MaxBackups = defaultFileMaxBackups
	}
	if file.MaxAgeDays == 0 {
		file.MaxAgeDays = defaultFileMaxAgeDays
	}

	rotatedFile := &lumberjack.Logger{
		Filename:   file.Path,
		MaxSize:    file.MaxSizeMB,
		MaxBackups: file.MaxBackups,
		MaxAge:     file.MaxAgeDays,
		Compress:   file.Compress,
		LocalTime:  file.LocalTime,
	}
	if output == nil {
		return rotatedFile, rotatedFile
	}
	return io.MultiWriter(output, rotatedFile), rotatedFile
}
