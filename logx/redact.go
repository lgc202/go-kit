package logx

import (
	"log/slog"
	"strings"
)

func redactAttr(options Options) func(groups []string, attr slog.Attr) slog.Attr {
	if len(options.RedactFields) == 0 {
		return nil
	}

	replacement := options.RedactValue
	if replacement == "" {
		replacement = defaultRedactValue
	}

	fields := make(map[string]struct{}, len(options.RedactFields))
	for _, field := range options.RedactFields {
		field = strings.ToLower(strings.TrimSpace(field))
		if field != "" {
			fields[field] = struct{}{}
		}
	}

	return func(groups []string, attr slog.Attr) slog.Attr {
		if _, ok := fields[strings.ToLower(attr.Key)]; !ok {
			return attr
		}
		return slog.String(attr.Key, replacement)
	}
}

func defaultRedactFields() []string {
	return []string{
		"access_token",
		"api_key",
		"authorization",
		"cookie",
		"password",
		"refresh_token",
		"secret",
		"session_id",
		"token",
	}
}
