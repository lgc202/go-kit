package examples

import (
	"context"
	"fmt"
	"os"

	"github.com/lgc202/go-kit/llm"
	"github.com/lgc202/go-kit/llm/provider/deepseek"
	"github.com/lgc202/go-kit/llm/schema"
)

// Reasoning demonstrates usage of deepseek-reasoner with thinking control.
func Reasoning() error {
	apiKey := os.Getenv("DEEPSEEK_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("DEEPSEEK_API_KEY environment variable is required")
	}

	client, err := deepseek.New(deepseek.Config{
		APIKey: apiKey,
		DefaultRequest: llm.RequestConfig{
			Model: "deepseek-reasoner",
		},
	})
	if err != nil {
		return err
	}

	// deepseek-reasoner returns reasoning_content in responses
	resp, err := client.Chat(context.Background(), []schema.Message{
		schema.UserMessage("If I have 5 apples and eat 2, then buy 3 more, how many do I have?"),
	},
		deepseek.WithThinking(true), // Enable reasoning mode (default for deepseek-reasoner)
		llm.WithMaxTokens(1000),
	)
	if err != nil {
		return err
	}

	if len(resp.Choices) > 0 {
		msg := resp.Choices[0].Message
		if msg.ReasoningContent != "" {
			fmt.Println("Reasoning:", msg.ReasoningContent)
		}
		fmt.Println("Answer:", msg.Text())
	}

	return nil
}

// ReasoningDisabled demonstrates disabling reasoning mode.
func ReasoningDisabled() error {
	apiKey := os.Getenv("DEEPSEEK_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("DEEPSEEK_API_KEY environment variable is required")
	}

	client, err := deepseek.New(deepseek.Config{
		APIKey: apiKey,
		DefaultRequest: llm.RequestConfig{
			Model: "deepseek-reasoner",
		},
	})
	if err != nil {
		return err
	}

	// Disable reasoning for faster, cheaper responses
	resp, err := client.Chat(context.Background(), []schema.Message{
		schema.UserMessage("What is the capital of France?"),
	},
		deepseek.WithThinking(false), // Disable reasoning
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

// ReasoningStreaming demonstrates streaming with separate reasoning callback.
func ReasoningStreaming() error {
	apiKey := os.Getenv("DEEPSEEK_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("DEEPSEEK_API_KEY environment variable is required")
	}

	client, err := deepseek.New(deepseek.Config{
		APIKey: apiKey,
		DefaultRequest: llm.RequestConfig{
			Model: "deepseek-reasoner",
		},
	})
	if err != nil {
		return err
	}

	resp, err := client.Chat(context.Background(), []schema.Message{
		schema.UserMessage("Solve: 15 * 23 - 47"),
	},
		llm.WithStreamingReasoningFunc(
			func(ctx context.Context, reasoningChunk, contentChunk []byte) error {
				if len(reasoningChunk) > 0 {
					fmt.Printf("\033[90m[Reasoning] %s\033[0m", reasoningChunk)
				}
				if len(contentChunk) > 0 {
					fmt.Printf("[Content] %s", contentChunk)
				}
				return nil
			},
		),
		deepseek.WithThinking(true),
	)
	if err != nil {
		return err
	}

	if len(resp.Choices) > 0 {
		fmt.Println("\nFinal:", resp.Choices[0].Message.Text())
	}

	return nil
}
