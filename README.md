# go-kit

Go 常用工具包集合。

| Package | 用途 |
|---|---|
| `config` | 配置加载：文件、环境变量、默认值、热更新 |
| `version` | 构建版本信息 |
| `logx` | 基于 `log/slog` 的日志初始化 |
| `redisx` | 基于 `go-redis` 的 Redis client 初始化 |
| `llm` | OpenAI Chat Completions 兼容 LLM client |

## Test

```bash
go test ./...
```
