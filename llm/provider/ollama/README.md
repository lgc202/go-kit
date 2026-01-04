# Ollama Provider

Ollama 本地模型 API 客户端

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

## 快速开始

```go
package main

import (
    "context"
    "fmt"
    "log"

    ollama "github.com/lgc202/go-kit/llm/provider/ollama/chat"
    "github.com/lgc202/go-kit/llm/schema"
)

func main() {
    client, err := ollama.New(ollama.Config{
        DefaultOptions: []llm.ChatOption{
            llm.WithModel("qwen2.5"),
        },
    })
    if err != nil {
        log.Fatal(err)
    }

    resp, err := client.Chat(context.Background(), []schema.Message{
        schema.UserMessage("什么是人工智能?"),
    })
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(resp.Choices[0].Message.Text())
}
```

## 选项配置

### 基础选项

```go
llm.WithTemperature(0.8)
llm.WithMaxCompletionTokens(4096)
llm.WithTopP(0.9)
```

### Ollama 特有选项

```go
// 设置模型保持加载时间
ollama.WithKeepAlive("30m")  // 30分钟
ollama.WithKeepAlive("24h")  // 24小时

// 设置模型运行选项
ollama.WithOptions(map[string]any{
    "temperature":     0.8,
    "top_k":           40,
    "num_ctx":         8192,
    "repeat_penalty":  1.1,
})

// 启用推理模式（用于 DeepSeek R1 等推理模型）
ollama.WithThink(true)
```

## Ollama 特有参数详解

### WithKeepAlive

设置模型在内存中保持加载的时间，避免每次请求都重新加载模型：

```go
ollama.WithKeepAlive("5m")   // 5分钟
ollama.WithKeepAlive("24h")  // 24小时
ollama.WithKeepAlive("")     // 使用默认值
```

### WithOptions

设置 Ollama 模型运行选项，这些选项直接传递给 Ollama：

```go
ollama.WithOptions(map[string]any{
    "temperature":     0.8,     // 采样温度
    "top_k":           40,      // 候选 token 数量
    "top_p":           0.9,     // 核采样阈值
    "num_ctx":         8192,    // 上下文窗口大小
    "num_predict":     4096,    // 最大生成 token 数
    "repeat_penalty":  1.1,     // 重复惩罚
    "stop":            []string{"\n", "User:"}, // 停止序列
    "mirostat":        2,       // Mirostat 采样
    "mirostat_tau":    5.0,     // Mirostat 目标熵
    "mirostat_eta":    0.1,     // Mirostat 学习率
})
```

参考文档: [Ollama Modelfile Parameters](https://github.com/ollama/ollama/blob/main/docs/modelfile.md#valid-parameters-and-values)

### WithFormat

设置结构化输出格式（JSON Schema 模式）：

```go
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

resp, err := client.Chat(ctx, []schema.Message{
    schema.UserMessage("提取: 张三今年25岁"),
},
    ollama.WithFormat(jsonSchema),
)
```

### WithThink

启用推理模式（用于 DeepSeek R1 等支持推理的模型）：

```go
ollama.WithThink(true)   // 启用推理
ollama.WithThink(false)  // 禁用推理
```

## 推理模式

使用 DeepSeek R1 等推理模型时，可以获取模型的推理过程：

```go
client, _ := ollama.New(ollama.Config{
    DefaultOptions: []llm.ChatOption{
        llm.WithModel("deepseek-r1"),
    },
})

resp, err := client.Chat(ctx, []schema.Message{
    schema.UserMessage("思考并回答: 如果我有 5 个苹果，吃了 2 个，又买了 3 个，现在有几个？"),
},
    ollama.WithThink(true),
)

msg := resp.Choices[0].Message
if msg.ReasoningContent != "" {
    fmt.Println("推理过程:", msg.ReasoningContent)
}
fmt.Println("答案:", msg.Text())
```

## JSON 结构化输出

```go
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

resp, err := client.Chat(ctx, []schema.Message{
    schema.UserMessage("提取信息: 张三今年25岁"),
},
    ollama.WithFormat(jsonSchema),
)
```

## 流式输出

### 基础流式

```go
stream, err := client.ChatStream(ctx, []schema.Message{
    schema.UserMessage("讲一个故事"),
})
defer stream.Close()

for {
    event, err := stream.Recv()
    if err != nil {
        break
    }
    if event.Type == schema.StreamEventDelta {
        fmt.Print(event.Delta)
    }
}
```

### 流式推理内容

```go
stream, err := client.ChatStream(ctx, []schema.Message{
    schema.UserMessage("计算：15 * 23 - 47"),
},
    ollama.WithThink(true),
)

for {
    event, err := stream.Recv()
    if err != nil {
        break
    }
    if len(event.Reasoning) > 0 {
        fmt.Printf("[思考] %s", event.Reasoning)
    }
    if len(event.Delta) > 0 {
        fmt.Printf("[内容] %s", event.Delta)
    }
}
```

## 配置

### 自定义 Base URL

```go
client, err := ollama.New(ollama.Config{
    BaseURL: "http://localhost:11434/v1",  // 默认值，可省略
})
```

### 自定义 HTTP 客户端

```go
httpClient := &http.Client{
    Timeout: 5 * time.Minute,  // 本地模型可能需要更长超时
}

client, err := ollama.New(ollama.Config{
    HTTPClient: httpClient,
})
```

### 客户端级默认配置

```go
client, err := ollama.New(ollama.Config{
    DefaultOptions: []llm.ChatOption{
        llm.WithModel("qwen2.5"),
        ollama.WithKeepAlive("30m"),
        ollama.WithOptions(map[string]any{
            "temperature": 0.8,
            "num_ctx":     8192,
        }),
    },
})
```

## 访问 Ollama 原生字段

Ollama 原生 API 返回的性能相关字段（如 `total_duration`、`load_duration`、`prompt_eval_count` 等）在使用 OpenAI 兼容端点时不可用。如需访问这些字段：

```go
resp, err := client.Chat(ctx, messages,
    llm.WithModel("qwen2.5"),
    llm.WithKeepRaw(true),  // 保留原始响应
)
// 使用 resp.Raw 访问 Ollama 原生 JSON 响应
```

## API 文档

详细 API 文档请参考: [Ollama API Documentation](https://github.com/ollama/ollama/blob/main/docs/api.md)
