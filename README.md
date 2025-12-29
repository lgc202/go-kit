# go-kit
golang 工具库

## Test

在受限环境（例如沙箱）中，默认的 Go build cache 目录可能不可写，导致 `go test ./...` 失败。

- 推荐：`make test`
- 或：`./scripts/test.sh`
