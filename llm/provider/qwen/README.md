# Qwen (通义千问) Provider

通义千问 DashScope API 客户端

## 安装

```bash
go get github.com/lgc202/go-kit/llm/provider/qwen
```

## 使用

### 基本用法

```go
package main

import (
    "context"
    "fmt"
    "log"

    qwen "github.com/lgc202/go-kit/llm/provider/qwen/chat"
    "github.com/lgc202/go-kit/llm/schema"
)

func main() {
    client, err := qwen.New(qwen.Config{
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

```go
import "github.com/lgc202/go-kit/llm"

resp, err := client.Chat(context.Background(), []schema.Message{
    schema.UserText("写一首诗"),
},
    llm.WithModel("qwen-plus"),
    llm.WithTemperature(0.7),
    llm.WithMaxCompletionTokens(4096),
)
```

### 启用深度思考模式

Qwen 支持深度思考模型，通过 `qwen.WithThinking()` 启用：

```go
import qwen "github.com/lgc202/go-kit/llm/provider/qwen/chat"

resp, err := client.Chat(context.Background(), []schema.Message{
    schema.UserText("解决这个数学问题: ..."),
},
    llm.WithModel("qwen-plus"),
    qwen.WithThinking(true),
)
```

也可以直接使用 `llm.WithExtraField()`：

```go
resp, err := client.Chat(context.Background(), messages,
    llm.WithModel("qwen-plus"),
    llm.WithExtraField("enable_thinking", true),
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
| qwen-turbo | 速度快，成本低 |
| qwen-plus | 性能均衡 |
| qwen-max | 性能最强 |
| qwen-long | 长上下文支持 |

## Qwen 特有参数

### WithThinking

启用或禁用深度思考模式

```go
qwen.WithThinking(true)  // 启用
qwen.WithThinking(false) // 禁用
```

参考文档: [https://www.alibabacloud.com/help/en/model-studio/deep-thinking](https://www.alibabacloud.com/help/en/model-studio/deep-thinking)

## API 文档

详细 API 文档请参考: [https://help.aliyun.com/zh/model-studio/developer-reference/use-qwen-by-calling-api](https://help.aliyun.com/zh/model-studio/developer-reference/use-qwen-by-calling-api)

## 配置选项

```go
type Config struct {
    BaseURL    string        // API 基础 URL，默认: https://dashscope.aliyuncs.com/compatible-mode/v1
    APIKey     string        // API 密钥 (必需)
    HTTPClient *http.Client  // 自定义 HTTP 客户端 (可选)

    // DefaultHeaders 请求级 headers 会覆盖这些默认值
    DefaultHeaders http.Header

    // DefaultOptions 客户端级别的默认请求选项
    DefaultOptions []llm.ChatOption
}
```
