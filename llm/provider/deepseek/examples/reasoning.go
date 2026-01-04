package examples

import (
	"context"
	"fmt"
	"os"

	"github.com/lgc202/go-kit/llm"
	deepseek "github.com/lgc202/go-kit/llm/provider/deepseek/chat"
	"github.com/lgc202/go-kit/llm/schema"
)

// Reasoning 演示 deepseek-reasoner 推理模式的用法
func Reasoning() error {
	apiKey := os.Getenv("DEEPSEEK_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("DEEPSEEK_API_KEY environment variable is required")
	}

	client, err := deepseek.New(deepseek.Config{
		APIKey:         apiKey,
		DefaultOptions: []llm.ChatOption{llm.WithModel("deepseek-reasoner")},
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

// ReasoningDisabled 演示禁用推理模式
func ReasoningDisabled() error {
	apiKey := os.Getenv("DEEPSEEK_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("DEEPSEEK_API_KEY environment variable is required")
	}

	client, err := deepseek.New(deepseek.Config{
		APIKey:         apiKey,
		DefaultOptions: []llm.ChatOption{llm.WithModel("deepseek-reasoner")},
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

// ReasoningStreaming 演示流式响应并区分推理内容和最终答案
func ReasoningStreaming() error {
	apiKey := os.Getenv("DEEPSEEK_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("DEEPSEEK_API_KEY environment variable is required")
	}

	client, err := deepseek.New(deepseek.Config{
		APIKey:         apiKey,
		DefaultOptions: []llm.ChatOption{llm.WithModel("deepseek-reasoner")},
	})
	if err != nil {
		return err
	}

	st, err := client.ChatStream(context.Background(), []schema.Message{
		schema.UserMessage("Solve: 15 * 23 - 47"),
	},
		deepseek.WithThinking(true),
	)
	if err != nil {
		return err
	}
	defer st.Close()

	for {
		ev, err := st.Recv()
		if err != nil {
			break
		}
		if len(ev.Reasoning) > 0 {
			fmt.Printf("\033[90m[Reasoning] %s\033[0m", ev.Reasoning)
		}
		if len(ev.Delta) > 0 {
			fmt.Printf("[Content] %s", ev.Delta)
		}
	}
	fmt.Println()

	return nil
}
