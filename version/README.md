# version

`version` 提供构建版本信息（gitVersion/gitCommit/buildDate 等）以及 CLI 友好的输出格式。

## 常见场景

### 1) 应用启动时打印版本号

```go
fmt.Println(version.Get().String())
```

### 2) CLI `version` 子命令（text/json/short）

示例见：

- `version/example`（flag 方式）
- `version/example/cobra`（cobra 集成）

### 3) 构建时注入版本信息（推荐）

构建脚本示例：`version/example/build.sh`

核心点是通过 `-ldflags -X` 覆盖包内变量：

```bash
go build -ldflags "\
  -X github.com/lgc202/go-kit/version.gitVersion=v1.2.3 \
  -X github.com/lgc202/go-kit/version.gitCommit=$(git rev-parse HEAD) \
  -X github.com/lgc202/go-kit/version.gitTreeState=clean \
  -X github.com/lgc202/go-kit/version.buildDate=$(date -u +'%Y-%m-%dT%H:%M:%SZ') \
"
```

## API 速查

- `version.Get() Info`
- `Info.String() string`：`v1.2.3` 或 `v1.2.3-dirty`
- `Info.ShortString() string`：仅版本号
- `Info.Text() string`：table 文本（适合终端）
- `Info.ToJSON()/ToJSONIndent()`

