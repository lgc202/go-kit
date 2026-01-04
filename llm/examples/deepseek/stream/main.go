package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/lgc202/go-kit/llm"
	deepseek "github.com/lgc202/go-kit/llm/provider/deepseek/chat"
	"github.com/lgc202/go-kit/llm/schema"
)

func main() {
	apiKey := os.Getenv("DEEPSEEK_API_KEY")
	if apiKey == "" {
		log.Fatal("DEEPSEEK_API_KEY environment variable is required")
	}

	client, err := deepseek.New(deepseek.Config{
		BaseConfig: deepseek.BaseConfig{
			APIKey: apiKey,
		},
		DefaultOptions: []llm.ChatOption{
			llm.WithModel("deepseek-chat"),
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	// 创建流式请求
	stream, err := client.ChatStream(context.Background(), []schema.Message{
		schema.UserMessage("用 3 点介绍 Go 语言的特点"),
	})
	if err != nil {
		log.Fatal(err)
	}
	defer stream.Close()

	fmt.Println("流式回复:")
	fmt.Println("---")

	for {
		event, err := stream.Recv()
		if err != nil {
			break
		}

		if event.Type == schema.StreamEventDelta {
			// 打印增量内容
			fmt.Print(event.Delta)
		}

		// 打印使用统计（在最后一个事件中）
		if event.Usage != nil {
			fmt.Printf("\n\n---\nToken 使用: %d\n", event.Usage.TotalTokens)
		}
	}

	fmt.Println("\n---")
	fmt.Println("流结束")
}
