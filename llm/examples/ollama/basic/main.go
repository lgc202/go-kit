package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/lgc202/go-kit/llm"
	ollama "github.com/lgc202/go-kit/llm/provider/ollama/chat"
	"github.com/lgc202/go-kit/llm/schema"
)

func main() {
	// 从环境变量获取模型名称，默认使用 qwen2.5
	modelName := os.Getenv("OLLAMA_MODEL")
	if modelName == "" {
		modelName = "qwen2.5"
	}

	// 创建 Ollama 客户端
	client, err := ollama.New(ollama.Config{
		DefaultOptions: []llm.ChatOption{
			llm.WithModel(modelName),
			llm.WithTemperature(0.8),
			ollama.WithThink(false),
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
		fmt.Printf("模型: %s\n", modelName)
		fmt.Println("回复:", resp.Choices[0].Message.Text())
	}

	// Token 使用统计
	fmt.Printf("\nToken 使用: 输入=%d, 输出=%d, 总计=%d\n",
		resp.Usage.PromptTokens,
		resp.Usage.CompletionTokens,
		resp.Usage.TotalTokens,
	)
}
