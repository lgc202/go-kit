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
	modelName := os.Getenv("MODEL")
	if modelName == "" {
		modelName = "deepseek-r1:1.5b"
	}

	// 使用 DeepSeek R1 推理模型
	client, err := ollama.New(ollama.Config{
		BaseConfig: ollama.BaseConfig{
			BaseURL: "http://localhost:11434/v1",
		},
		DefaultOptions: []llm.ChatOption{
			llm.WithModel(modelName),
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	// 使用推理模型解决问题
	resp, err := client.Chat(context.Background(), []schema.Message{
		schema.UserMessage("一个房间里有 3 个灯泡，房间外有 3 个开关。你只能进入房间一次，如何确定哪个开关控制哪个灯泡？请一步步推理。"),
	},
		ollama.WithThink(true), // 启用推理模式
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
}
