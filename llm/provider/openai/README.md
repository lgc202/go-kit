# OpenAI Provider

go-kit 的企业级 OpenAI API 客户端。

## 快速开始

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/lgc202/go-kit/llm"
    openai "github.com/lgc202/go-kit/llm/provider/openai/chat"
    "github.com/lgc202/go-kit/llm/schema"
)

func main() {
    client, err := openai.New(openai.Config{
        APIKey: os.Getenv("OPENAI_API_KEY"),
        DefaultOptions: []llm.ChatOption{
            llm.WithModel("gpt-4o-mini"),
        },
    })
    if err != nil {
        panic(err)
    }

    resp, err := client.Chat(context.Background(), []schema.Message{
        schema.UserMessage("什么是 Go 语言？"),
    })
    if err != nil {
        panic(err)
    }

    fmt.Println(resp.Choices[0].Message.Text())
}
```

## 选项配置

### 基础选项

```go
llm.WithModel("gpt-4o-mini")
llm.WithTemperature(0.7)
llm.WithMaxCompletionTokens(1000)
llm.WithTopP(0.9)
llm.WithFrequencyPenalty(0.5)
llm.WithPresencePenalty(0.5)
llm.WithStop([]string{"\n", "END"})
```

### 工具调用

```go
llm.WithTools(tool1, tool2)
llm.WithToolChoice(schema.ToolChoice{Mode: schema.ToolChoiceAuto})  // 自动决定
llm.WithToolChoice(schema.ToolChoice{Mode: schema.ToolChoiceNone})  // 禁用工具
llm.WithParallelToolCalls(true)  // 允许并行调用
```

### 响应格式

```go
// JSON 对象模式
llm.WithResponseFormat(schema.ResponseFormat{Type: "json_object"})

// JSON Schema 模式
llm.WithResponseFormat(schema.ResponseFormat{
    Type:       "json_schema",
    JSONSchema: schemaJSON,
})
```

## 多模态输入

GPT-4o 支持图片和文本混合输入：

```go
messages := []schema.Message{
    schema.Message{
        Role: schema.RoleUser,
        Content: []schema.ContentPart{
            schema.TextPart("这张图片里有什么？"),
            schema.ImageURLPart("https://example.com/image.jpg"),
        },
    },
}

resp, err := client.Chat(ctx, messages)
```

## 工具/函数调用

```go
weatherTool := schema.Tool{
    Type: schema.ToolTypeFunction,
    Function: schema.FunctionDefinition{
        Name:        "get_weather",
        Description: "获取指定地点的当前天气",
        Parameters:  schema.MustJSON(map[string]any{
            "type": "object",
            "properties": map[string]any{
                "location": map[string]any{
                    "type":        "string",
                    "description": "城市名称",
                },
            },
            "required": []string{"location"},
        }),
    },
}

resp, err := client.Chat(ctx, []schema.Message{
    schema.UserMessage("北京今天天气怎么样？"),
},
    llm.WithTools(weatherTool),
)

msg := resp.Choices[0].Message
if len(msg.ToolCalls) > 0 {
    for _, tc := range msg.ToolCalls {
        fmt.Printf("调用: %s\n", tc.Function.Name)
        fmt.Printf("参数: %s\n", tc.Function.Arguments)

        // 执行工具并将结果返回
        result := `{"temp": "22°C", "condition": "晴朗"}`
        resp2, _ := client.Chat(ctx, []schema.Message{
            schema.UserMessage("北京今天天气怎么样？"),
            msg,
            schema.ToolResultMessage(tc.ID, result),
        })
        fmt.Println(resp2.Choices[0].Message.Text())
    }
}
```

## JSON 模式

### JSON 对象响应

```go
resp, err := client.Chat(ctx, []schema.Message{
    schema.UserMessage("列出 3 种编程语言，以 JSON 格式返回"),
},
    llm.WithResponseFormat(schema.ResponseFormat{Type: "json_object"}),
)
fmt.Println(resp.Choices[0].Message.Text())
```

### JSON Schema 响应

```go
schemaJSON := schema.MustJSON(map[string]any{
    "type": "object",
    "properties": map[string]any{
        "name": map[string]any{"type": "string"},
        "year": map[string]any{"type": "integer"},
    },
    "required": []string{"name", "year"},
})

resp, err := client.Chat(ctx, []schema.Message{
    schema.UserMessage("介绍一下 Go 语言"),
},
    llm.WithResponseFormat(schema.ResponseFormat{
        Type:       "json_schema",
        JSONSchema: schemaJSON,
    }),
)
fmt.Println(resp.Choices[0].Message.Text())
```

## 流式输出

### 基础流式

```go
stream, err := client.ChatStream(ctx, []schema.Message{
    schema.UserMessage("从 1 数到 10"),
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

### 流式输出包含使用统计

```go
stream, err := client.ChatStream(ctx, []schema.Message{
    schema.UserMessage("你好"),
},
    llm.WithStreamIncludeUsage(),
)

for {
    event, err := stream.Recv()
    if err != nil {
        break
    }
    if event.Type == schema.StreamEventDelta {
        fmt.Print(event.Delta)
    }
    if event.Usage != nil {
        fmt.Printf("\nTokens: %d\n", event.Usage.TotalTokens)
    }
}
```

## Token 使用统计

```go
resp, err := client.Chat(...)
usage := resp.Usage

fmt.Println("输入 token:", usage.PromptTokens)
fmt.Println("输出 token:", usage.CompletionTokens)
fmt.Println("总 token:", usage.TotalTokens)
```

## 配置

### 客户端级默认配置

```go
client, err := openai.New(openai.Config{
    APIKey: os.Getenv("OPENAI_API_KEY"),
    DefaultOptions: []llm.ChatOption{
        llm.WithModel("gpt-4o-mini"),
        llm.WithTemperature(0.7),
    },
})
```

### 自定义 Base URL

```go
client, err := openai.New(openai.Config{
    BaseURL: "https://your-proxy.example.com/v1",
    APIKey:  os.Getenv("OPENAI_API_KEY"),
})
```

### 自定义 HTTP 客户端

```go
httpClient := &http.Client{
    Timeout: 30 * time.Second,
}

client, err := openai.New(openai.Config{
    APIKey:     os.Getenv("OPENAI_API_KEY"),
    HTTPClient: httpClient,
})
```

### 默认请求头

```go
headers := make(http.Header)
headers.Set("X-Custom-Header", "value")

client, err := openai.New(openai.Config{
    APIKey:         os.Getenv("OPENAI_API_KEY"),
    DefaultHeaders: headers,
})
```

## API 参考

- [OpenAI API 文档](https://platform.openai.com/docs/api-reference)
- [llm 包](../../README.md)
