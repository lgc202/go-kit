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

	// 创建客户端，配置默认模型
	client, err := deepseek.New(deepseek.Config{
		BaseConfig: deepseek.BaseConfig{
			APIKey: apiKey,
		},
		DefaultOptions: []llm.ChatOption{
			llm.WithModel("deepseek-chat"),
			llm.WithTemperature(0.7),
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	// 基础对话
	resp, err := client.Chat(context.Background(), []schema.Message{
		schema.SystemMessage("You are a helpful assistant."),
		schema.UserMessage("什么是 Go 语言？用一句话概括。"),
	})
	if err != nil {
		log.Fatal(err)
	}

	if len(resp.Choices) > 0 {
		fmt.Println("回复:", resp.Choices[0].Message.Text())
	}

	// Token 使用统计
	fmt.Printf("\nToken 使用: 输入=%d, 输出=%d, 总计=%d\n",
		resp.Usage.PromptTokens,
		resp.Usage.CompletionTokens,
		resp.Usage.TotalTokens,
	)
}
