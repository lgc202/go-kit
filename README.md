# go-kit

`github.com/lgc202/go-kit` 是一个 Go 通用工具箱仓库：尽量用“小而稳”的包来解决工程里反复出现的问题（配置、版本信息、HTTP、LLM 等），并提供可直接拷走的示例。

## Requirements

- Go: 见 `go.mod` 的 `go` 版本（当前为 `go 1.25.5`）

## Packages

| Package | 用途 | 入口文档 |
|---|---|---|
| `config` | 类型安全的配置加载（文件 + env + 默认值），支持热更新回调 | `config/README.md` |
| `version` | 构建时注入版本信息，支持 text/json/short 输出（CLI 友好） | `version/README.md` |
| `httpx` | 企业级 HTTP Client：BaseURL、默认头、RequestID、重试、错误模型、Hook | `httpx/README.md` |
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

### httpx: 统一出站 HTTP（重试/RequestID/错误体截断）

```go
client, _ := httpx.New(
	httpx.WithBaseURL("https://api.example.com"),
	httpx.WithTimeout(30*time.Second),
	httpx.WithDefaultHeader("Accept", "application/json"),
)

req, _ := client.NewJSONRequest(ctx, http.MethodPost, "/v1/users", map[string]any{
	"name": "alice",
})
resp, err := client.DoStatus(req) // 非 2xx -> *httpx.Error
_ = resp
_ = err
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

# httpx
go run ./httpx/examples/basic
go run ./httpx/examples/retry
go run ./httpx/examples/observability
```

## Dev / Test

推荐（可在受限环境下把 cache 写到本地目录）：

```bash
make test
# 或
./scripts/test.sh
```
