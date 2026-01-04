# Kimi (Moonshot AI) Provider

go-kit 的企业级 Kimi (Moonshot AI) API 客户端。

## 快速开始

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/lgc202/go-kit/llm"
    kimi "github.com/lgc202/go-kit/llm/provider/kimi/chat"
    "github.com/lgc202/go-kit/llm/schema"
)

func main() {
    client, err := kimi.New(kimi.Config{
        APIKey: os.Getenv("MOONSHOT_API_KEY"),
        DefaultOptions: []llm.ChatOption{
            llm.WithModel("moonshot-v1-8k"),
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
llm.WithTemperature(0.7)
llm.WithMaxCompletionTokens(4096)
llm.WithTopP(0.9)
```

### 工具调用

```go
llm.WithTools(tool1, tool2)
llm.WithToolChoice(schema.ToolChoice{Mode: schema.ToolChoiceAuto})
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

messages := []schema.Message{
    schema.UserMessage("北京今天天气怎么样？"),
}

// 循环处理：模型可能需要多次调用工具
const maxSteps = 8
for step := 0; step < maxSteps; step++ {
    resp, err := client.Chat(ctx, messages, llm.WithTools(weatherTool))
    if err != nil {
        break
    }

    msg := resp.Choices[0].Message
    messages = append(messages, msg)  // 重要：添加 assistant 消息（包含 tool_calls）到历史

    if len(msg.ToolCalls) == 0 {
        // 模型不再调用工具，返回最终回复
        fmt.Println(msg.Text())
        break
    }

    // 执行工具调用
    for _, tc := range msg.ToolCalls {
        fmt.Printf("调用: %s\n", tc.Function.Name)
        fmt.Printf("参数: %s\n", tc.Function.Arguments)

        // 执行工具获取结果并添加到历史
        result := getWeather(tc.Function.Arguments)
        messages = append(messages, schema.ToolResultMessage(tc.ID, result))
    }
}
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

### 流式输出包含使用统计

```go
stream, err := client.ChatStream(ctx, []schema.Message{
    schema.UserMessage("你好"),
},
    llm.WithStreamIncludeUsage(),
)
```

## 配置

### 客户端级默认配置

```go
client, err := kimi.New(kimi.Config{
    APIKey: os.Getenv("MOONSHOT_API_KEY"),
    DefaultOptions: []llm.ChatOption{
        llm.WithModel("moonshot-v1-8k"),
        llm.WithTemperature(0.7),
    },
})
```

### 自定义 Base URL

```go
client, err := kimi.New(kimi.Config{
    BaseURL: "https://your-proxy.example.com",
    APIKey:  os.Getenv("MOONSHOT_API_KEY"),
})
```

### 自定义 HTTP 客户端

```go
httpClient := &http.Client{
    Timeout: 30 * time.Second,
}

client, err := kimi.New(kimi.Config{
    APIKey:     os.Getenv("MOONSHOT_API_KEY"),
    HTTPClient: httpClient,
})
```

### 默认请求头

```go
headers := make(http.Header)
headers.Set("X-Custom-Header", "value")

client, err := kimi.New(kimi.Config{
    APIKey:         os.Getenv("MOONSHOT_API_KEY"),
    DefaultHeaders: headers,
})
```

## API 文档

- [Kimi API 文档](https://platform.moonshot.cn/docs)
- [llm 包](../../README.md)
