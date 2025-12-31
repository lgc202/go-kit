package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/lgc202/go-kit/llm"
	"github.com/lgc202/go-kit/llm/provider/ollama"
	"github.com/lgc202/go-kit/llm/schema"
)

func main() {
	modelName := os.Getenv("OLLAMA_MODEL")
	if modelName == "" {
		modelName = "deepseek-r1:1.5b"
	}

	m, err := ollama.New(ollama.Config{
		DefaultOptions: []llm.RequestOption{llm.WithModel(modelName)},
	})
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	messages := []schema.Message{
		schema.SystemMessage("You are a helpful assistant."),
		schema.UserMessage("用一句话介绍一下 Go 语言。"),
	}

	resp, err := m.Chat(ctx, messages, llm.WithTemperature(0.7))
	if err != nil {
		log.Fatal(err)
	}

	if len(resp.Choices) == 0 {
		log.Fatal("empty response")
	}
	fmt.Println(resp.Choices[0].Message.Text())
}
