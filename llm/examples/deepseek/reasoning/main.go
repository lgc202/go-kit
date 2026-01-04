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

	// 创建推理模型客户端
	client, err := deepseek.New(deepseek.Config{
		BaseConfig: deepseek.BaseConfig{
			APIKey: apiKey,
		},
		DefaultOptions: []llm.ChatOption{
			llm.WithModel("deepseek-reasoner"),
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	// 使用推理模型解决数学问题
	resp, err := client.Chat(context.Background(), []schema.Message{
		schema.UserMessage("如果我有 5 个苹果，吃了 2 个，又买了 3 个，又吃掉 1 个，现在有几个？请一步步推理。"),
	},
		deepseek.WithThinking(true), // 启用推理模式
	)
	if err != nil {
		log.Fatal(err)
	}

	msg := resp.Choices[0].Message

	// 打印推理过程
	if msg.ReasoningContent != "" {
		fmt.Println("=== 推理过程 ===")
		fmt.Println(msg.ReasoningContent)
		fmt.Println()
	}

	// 打印最终答案
	fmt.Println("=== 最终答案 ===")
	fmt.Println(msg.Text())

	// 推理 token 统计
	if resp.Usage.CompletionTokensDetails != nil {
		fmt.Printf("\n推理 Token: %d\n", resp.Usage.CompletionTokensDetails.ReasoningTokens)
	}
}
