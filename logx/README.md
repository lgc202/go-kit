# logx

`logx` 是基于标准库 `log/slog` 的日志初始化包。它不重新设计日志接口，
业务代码仍然使用 `slog.Logger` 的 `Info`、`Error` 等方法。

## 功能

- JSON / text 输出
- stdout / 自定义 `io.Writer` / 文件输出
- 文件日志轮转
- 运行时修改日志级别
- 默认服务字段：`service`、`env`、`version`
- 按字段名脱敏

## 快速开始

```go
import "log/slog"

logger, err := logx.New(logx.Options{
	Format:  logx.FormatJSON,
	Level:   logx.LevelInfo,
	Service: "user-api",
	Env:     "prod",
	Version: "v1.2.3",
})
if err != nil {
	return err
}

logger.Info("server started", slog.String("addr", ":8080"))

_ = logger.SetLevel(logx.LevelDebug)
```

## 文件轮转

默认写 stdout。需要写文件时，设置 `File.Path`。如果同时设置 `Output`，
日志会同时写入 `Output` 和轮转文件。

```go
logger, err := logx.New(logx.Options{
	Format: logx.FormatJSON,
	File: logx.FileOptions{
		Path:       "./logs/app.log",
		MaxSizeMB:  100,
		MaxBackups: 10,
		MaxAgeDays: 7,
		Compress:   true,
	},
})
if err != nil {
	return err
}
defer logger.Close()

logger.Info("file logging enabled")
```

## 脱敏

默认会脱敏常见敏感字段，例如 `token`、`authorization`、`password`、
`secret`、`cookie`。

```go
logger, err := logx.New(logx.Options{
	RedactFields: []string{"token", "authorization", "private_key"},
})
```
