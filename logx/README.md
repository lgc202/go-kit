# logx

基于 `log/slog` 的日志初始化包，支持动态级别、脱敏和文件轮转。

```go
logger, err := logx.New(
	logx.WithFormat(logx.FormatJSON),
	logx.WithService("user-api"),
)
if err != nil {
	return err
}
defer logger.Close()

logger.Info("server started")
_ = logger.SetLevel(logx.LevelDebug)
```

文件输出：

```go
logger, err := logx.New(
	logx.WithFile(logx.FileOptions{Path: "./logs/app.log"}),
)
```
