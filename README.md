# go-kit

`github.com/lgc202/go-kit` 是一个 Go 通用工具箱仓库：尽量用“小而稳”的包来解决工程里反复出现的问题（配置、版本信息、日志、Redis、LLM 等），并提供可直接拷走的示例。

## Requirements

- Go: 见 `go.mod` 的 `go` 版本（当前为 `go 1.25.5`）

## Packages

| Package | 用途 | 入口文档 |
|---|---|---|
| `config` | 类型安全的配置加载（文件 + env + 默认值），支持热更新回调 | `config/README.md` |
| `version` | 构建时注入版本信息，支持 text/json/short 输出（CLI 友好） | `version/README.md` |
| `logx` | 基于 `log/slog` 的结构化日志：动态级别、脱敏、文件轮转 | `logx/README.md` |
| `redisx` | 基于 `go-redis` 的 Redis client 初始化：单机、Sentinel、Cluster | `redisx/README.md` |
| `llm` | 统一的 LLM Client（OpenAI Chat Completions 兼容格式）+ 多 provider | `llm/README.md` |

## Quick Start

### config: 文件 + env + 热更新

```go
type AppConfig struct {
	Server struct {
		Host string `mapstructure:"host"`
		Port int    `mapstructure:"port"`
	} `mapstructure:"server"`
}

cfg, err := config.Load[AppConfig]("./config.yaml",
	config.WithDefaults[AppConfig](map[string]any{
		"server.host": "0.0.0.0",
		"server.port": 8080,
	}),
	config.WithEnv[AppConfig]("APP"), // APP_SERVER_HOST / APP_SERVER_PORT
)
if err != nil { /* ... */ }

cfg.OnChange(func(old, new AppConfig) {
	if config.Changed(old.Server, new.Server) {
		// reload server...
	}
})
```

### version: CLI 里输出版本信息

```go
info := version.Get()
fmt.Println(info.Text())        // table
fmt.Println(info.ShortString()) // v1.2.3
```

> 构建时注入 `-ldflags` 的示例见 `version/example/build.sh`。

### logx: 结构化日志（动态级别/脱敏/文件轮转）

```go
import "log/slog"

logger, err := logx.New(
	logx.WithFormat(logx.FormatJSON),
	logx.WithLevel(logx.LevelInfo),
	logx.WithService("user-api"),
	logx.WithEnv("prod"),
)
if err != nil { /* ... */ }

logger.Info("server started", slog.String("addr", ":8080"))
_ = logger.SetLevel(logx.LevelDebug)
```

### redisx: Redis client 初始化

```go
client, err := redisx.New(redisx.WithAddr("127.0.0.1:6379"))
if err != nil { /* ... */ }
defer client.Close()
```

### llm: 统一 Chat / Stream / Tools

`llm/README.md` 有完整示例与 provider 列表（OpenAI / DeepSeek / Kimi / Qwen / Ollama）。

## Examples

```bash
# config
go run ./config/examples/yaml
go run ./config/examples/json

# version
go run ./version/example -version
go run ./version/example/cobra version -o json

# llm
go run ./llm/examples/ollama/basic

```

## Dev / Test

推荐（可在受限环境下把 cache 写到本地目录）：

```bash
go test ./...
```
