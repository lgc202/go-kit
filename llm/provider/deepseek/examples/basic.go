// Package examples provides usage examples for the DeepSeek provider.
package examples

import (
	"context"
	"fmt"
	"os"

	"github.com/lgc202/go-kit/llm"
	"github.com/lgc202/go-kit/llm/provider/deepseek"
	"github.com/lgc202/go-kit/llm/schema"
)

// Basic demonstrates basic usage of the DeepSeek provider.
func Basic() error {
	apiKey := os.Getenv("DEEPSEEK_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("DEEPSEEK_API_KEY environment variable is required")
	}

	client, err := deepseek.New(deepseek.Config{
		APIKey:         apiKey,
		DefaultOptions: []llm.RequestOption{llm.WithModel("deepseek-chat")},
	})
	if err != nil {
		return err
	}

	resp, err := client.Chat(context.Background(), []schema.Message{
		schema.SystemMessage("You are a helpful assistant."),
		schema.UserMessage("What is 2+2?"),
	},
		llm.WithTemperature(0.7),
		llm.WithMaxTokens(100),
	)
	if err != nil {
		return err
	}

	if len(resp.Choices) > 0 {
		fmt.Println(resp.Choices[0].Message.Text())
	}

	return nil
}

// Streaming demonstrates streaming usage.
func Streaming() error {
	apiKey := os.Getenv("DEEPSEEK_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("DEEPSEEK_API_KEY environment variable is required")
	}

	client, err := deepseek.New(deepseek.Config{
		APIKey:         apiKey,
		DefaultOptions: []llm.RequestOption{llm.WithModel("deepseek-chat")},
	})
	if err != nil {
		return err
	}

	stream, err := client.ChatStream(context.Background(), []schema.Message{
		schema.UserMessage("Count from 1 to 10"),
	})
	if err != nil {
		return err
	}
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
	fmt.Println()

	return nil
}
