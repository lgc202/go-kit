package main

import (
	"context"
	"fmt"
	"os"

	"github.com/lgc202/go-kit/llm"
	"github.com/lgc202/go-kit/llm/providers/openai"
)

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("set OPENAI_API_KEY")
		return
	}

	provider, err := openai.New(apiKey, openai.WithDefaultModel("gpt-4o-mini"))
	if err != nil {
		panic(err)
	}
	client := llm.New(provider)

	stream, err := client.ChatStream(context.Background(), []llm.Message{llm.User("Say hello.")})
	if err != nil {
		panic(err)
	}
	defer stream.Close()

	for {
		ev, err := stream.Recv()
		if err != nil {
			break
		}
		if ev.Kind == llm.StreamEventPartDelta && ev.PartDelta != nil && ev.PartDelta.Type == llm.ContentPartText {
			fmt.Print(ev.PartDelta.TextDelta)
		}
		if ev.Done() {
			fmt.Println()
			break
		}
	}
}
