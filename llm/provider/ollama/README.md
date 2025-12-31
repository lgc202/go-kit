# Ollama Provider

Ollama 本地模型 API 客户端

## 安装

```bash
go get github.com/lgc202/go-kit/llm/provider/ollama
```

## 前置要求

需要先安装并运行 Ollama：

```bash
# macOS/Linux
curl -fsSL https://ollama.com/install.sh | sh

# 或使用 Docker
docker run -d -v ollama:/root/.ollama -p 11434:11434 --name ollama ollama/ollama

# 拉取模型
ollama pull qwen2.5
```

## 使用

### 基本用法

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/lgc202/go-kit/llm/provider/ollama"
    "github.com/lgc202/go-kit/llm/schema"
)

func main() {
    client, err := ollama.New(ollama.Config{
        // 本地 Ollama 默认地址，可省略
        // BaseURL: "http://localhost:11434/v1",
    })
    if err != nil {
        log.Fatal(err)
    }

    resp, err := client.Chat(context.Background(), []schema.Message{
        schema.UserText("什么是人工智能?"),
    },
        llm.WithModel("qwen2.5"),
    )
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(resp.Choices[0].Message.Text())
}
```

### 使用 Ollama 特有选项

```go
import "github.com/lgc202/go-kit/llm/provider/ollama"

resp, err := client.Chat(context.Background(), []schema.Message{
    schema.UserText("讲一个故事"),
},
    llm.WithModel("qwen2.5"),
    // 设置模型保持加载时间
    ollama.WithKeepAlive("30m"),
    // 设置 Ollama 运行选项
    ollama.WithOptions(map[string]any{
        "temperature": 0.8,
        "top_k":       40,
        "num_ctx":     8192,
    }),
)
```

### JSON 结构化输出

```go
// 定义 JSON Schema
jsonSchema := map[string]any{
    "type": "object",
    "properties": map[string]any{
        "name": map[string]any{
            "type":        "string",
            "description": "人名",
        },
        "age": map[string]any{
            "type":        "integer",
            "description": "年龄",
        },
    },
    "required": []string{"name", "age"},
}

resp, err := client.Chat(context.Background(), []schema.Message{
    schema.UserText("提取: 张三今年25岁"),
},
    llm.WithModel("qwen2.5"),
    ollama.WithFormat(jsonSchema),
)
```

### 启用推理模式

```go
resp, err := client.Chat(context.Background(), []schema.Message{
    schema.UserText("思考并回答: ..."),
},
    llm.WithModel("qwen2.5:7b"),
    ollama.WithThink(true),
)
```

### 访问 Ollama 特有字段

Ollama 原生 API 返回的性能相关字段（如 `total_duration`, `load_duration`, `prompt_eval_count` 等）在使用 OpenAI 兼容端点时不可用。如需访问这些字段，可以：

```go
resp, err := client.Chat(context.Background(), messages,
    llm.WithModel("qwen2.5"),
    llm.WithKeepRaw(true), // 保留原始响应
)
// 使用 resp.Raw 访问 Ollama 原生 JSON 响应
```

### 流式响应

```go
stream, err := client.ChatStream(context.Background(), []schema.Message{
    schema.UserText("讲一个故事"),
},
    llm.WithModel("qwen2.5"),
)
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

## Ollama 特有参数

### WithFormat

设置结构化输出格式（JSON Schema 模式）

```go
ollama.WithFormat(jsonSchema map[string]any)
```

### WithKeepAlive

设置模型在内存中保持加载的时间

```go
ollama.WithKeepAlive("5m")   // 5分钟
ollama.WithKeepAlive("24h")  // 24小时
ollama.WithKeepAlive("")     // 使用默认值
```

### WithOptions

设置 Ollama 模型运行选项

```go
ollama.WithOptions(map[string]any{
    "temperature":     0.8,    // 采样温度
    "top_k":           40,     // 候选 token 数量
    "top_p":           0.9,    // 核采样阈值
    "num_ctx":         8192,   // 上下文窗口大小
    "num_predict":     4096,   // 最大生成 token 数
    "repeat_penalty":  1.1,    // 重复惩罚
    "stop":            []string{"\n", "User:"}, // 停止序列
    "mirostat":        2,      // Mirostat 采样
    "mirostat_tau":    5.0,    // Mirostat 目标熵
    "mirostat_eta":    0.1,    // Mirostat 学习率
})
```

参考文档: [https://github.com/ollama/ollama/blob/main/docs/modelfile.md#valid-parameters-and-values](https://github.com/ollama/ollama/blob/main/docs/modelfile.md#valid-parameters-and-values)

### WithThink

启用推理模式（用于支持推理的模型）

```go
ollama.WithThink(true)   // 启用推理
ollama.WithThink(false)  // 禁用推理
```

## 常用模型

| 模型名称 | 描述 |
|---------|------|
| qwen2.5 | 通义千问 2.5 |
| llama3.2 | Llama 3.2 |
| mistral | Mistral 7B |
| deepseek-r1 | DeepSeek R1 (支持推理) |
| qwen2.5-coder | Qwen 2.5 Coder |

## API 文档

详细 API 文档请参考: [https://github.com/ollama/ollama/blob/main/docs/api.md](https://github.com/ollama/ollama/blob/main/docs/api.md)

## 配置选项

```go
type Config struct {
    BaseURL    string        // API 基础 URL，默认: http://localhost:11434/v1
    APIKey     string        // API 密钥 (Ollama 通常不需要)
    HTTPClient *http.Client  // 自定义 HTTP 客户端 (可选)

    // DefaultHeaders 请求级 headers 会覆盖这些默认值
    DefaultHeaders http.Header

    // DefaultOptions 客户端级别的默认请求选项
    DefaultOptions []llm.RequestOption
}
```
