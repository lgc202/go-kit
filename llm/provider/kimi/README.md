# Kimi (Moonshot AI) Provider

Kimi (Moonshot AI) API 客户端

## 安装

```bash
go get github.com/lgc202/go-kit/llm/provider/kimi
```

## 使用

### 基本用法

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/lgc202/go-kit/llm/provider/kimi"
    "github.com/lgc202/go-kit/llm/schema"
)

func main() {
    client, err := kimi.New(kimi.Config{
        APIKey: "your-api-key",
    })
    if err != nil {
        log.Fatal(err)
    }

    resp, err := client.Chat(context.Background(), []schema.Message{
        schema.UserText("什么是人工智能?"),
    })
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(resp.Choices[0].Message.Text())
}
```

### 使用请求选项

Kimi 支持所有标准 OpenAI 兼容参数，通过 `llm` 包的选项函数设置：

```go
import "github.com/lgc202/go-kit/llm"

resp, err := client.Chat(context.Background(), []schema.Message{
    schema.UserText("写一首诗"),
},
    llm.WithModel("moonshot-v1-8k"),
    llm.WithTemperature(0.7),
    llm.WithMaxCompletionTokens(4096),
    llm.WithTopP(0.9),
)
```

### 流式响应

```go
stream, err := client.ChatStream(context.Background(), []schema.Message{
    schema.UserText("讲一个故事"),
})
if err != nil {
    log.Fatal(err)
}

for stream.Next() {
    event := stream.Event()
    if event.Type == llm.StreamEventDelta {
        fmt.Print(event.Delta)
    }
}
if err := stream.Err(); err != nil {
    log.Fatal(err)
}
```

## 支持的模型

| 模型名称 | 描述 |
|---------|------|
| moonshot-v1-8k | 8K 上下文窗口 |
| moonshot-v1-32k | 32K 上下文窗口 |
| moonshot-v1-128k | 128K 上下文窗口 |

## API 文档

详细 API 文档请参考: [https://platform.moonshot.cn/docs](https://platform.moonshot.cn/docs)

## 配置选项

```go
type Config struct {
    BaseURL    string        // API 基础 URL，默认: https://api.moonshot.cn/v1
    APIKey     string        // API 密钥 (必需)
    HTTPClient *http.Client  // 自定义 HTTP 客户端 (可选)

    // DefaultHeaders 请求级 headers 会覆盖这些默认值
    DefaultHeaders http.Header

    // DefaultOptions 客户端级别的默认请求选项
    DefaultOptions []llm.RequestOption
}
```
