package examples

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/lgc202/go-kit/llm"
	deepseek "github.com/lgc202/go-kit/llm/provider/deepseek/chat"
	"github.com/lgc202/go-kit/llm/schema"
)

// ToolCalling demonstrates function calling with DeepSeek.
func ToolCalling() error {
	apiKey := os.Getenv("DEEPSEEK_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("DEEPSEEK_API_KEY environment variable is required")
	}

	client, err := deepseek.New(deepseek.Config{
		APIKey:         apiKey,
		DefaultOptions: []llm.ChatOption{llm.WithModel("deepseek-chat")},
	})
	if err != nil {
		return err
	}

	// Define a tool (function)
	weatherTool := schema.Tool{
		Type: schema.ToolTypeFunction,
		Function: schema.FunctionDefinition{
			Name:        "get_weather",
			Description: "Get the current weather for a location",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"location": {
						"type": "string",
						"description": "The city and state, e.g. San Francisco, CA"
					}
				},
				"required": ["location"]
			}`),
		},
	}

	resp, err := client.Chat(context.Background(), []schema.Message{
		schema.UserMessage("What's the weather in Tokyo?"),
	},
		llm.WithTools(weatherTool),
		llm.WithMaxTokens(200),
	)
	if err != nil {
		return err
	}

	if len(resp.Choices) > 0 {
		msg := resp.Choices[0].Message
		if len(msg.ToolCalls) > 0 {
			// Model wants to call a tool
			for _, tc := range msg.ToolCalls {
				fmt.Printf("Tool call: %s\n", tc.Function.Name)
				fmt.Printf("Arguments: %s\n", tc.Function.Arguments)

				// Execute the tool and send result back
				toolResult := `{"temperature": "22°C", "condition": "sunny"}`
				resp2, err := client.Chat(context.Background(), []schema.Message{
					schema.UserMessage("What's the weather in Tokyo?"),
					msg,
					schema.ToolResultMessage(tc.ID, toolResult),
				},
					llm.WithTools(weatherTool),
				)
				if err != nil {
					return err
				}
				if len(resp2.Choices) > 0 {
					fmt.Println("Response:", resp2.Choices[0].Message.Text())
				}
			}
		} else {
			fmt.Println("Response:", msg.Text())
		}
	}

	return nil
}

// ToolChoiceRequired demonstrates forcing the model to call a tool.
func ToolChoiceRequired() error {
	apiKey := os.Getenv("DEEPSEEK_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("DEEPSEEK_API_KEY environment variable is required")
	}

	client, err := deepseek.New(deepseek.Config{
		APIKey:         apiKey,
		DefaultOptions: []llm.ChatOption{llm.WithModel("deepseek-chat")},
	})
	if err != nil {
		return err
	}

	calculatorTool := schema.Tool{
		Type: schema.ToolTypeFunction,
		Function: schema.FunctionDefinition{
			Name:        "calculate",
			Description: "Perform a mathematical calculation",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"expression": {
						"type": "string",
						"description": "The mathematical expression to evaluate"
					}
				},
				"required": ["expression"]
			}`),
		},
	}

	resp, err := client.Chat(context.Background(), []schema.Message{
		schema.UserMessage("Hello"),
	},
		llm.WithTools(calculatorTool),
		llm.WithToolChoice(schema.ToolChoice{Mode: "required"}), // Force tool call
	)
	if err != nil {
		return err
	}

	if len(resp.Choices) > 0 {
		fmt.Println("Tool calls:", resp.Choices[0].Message.ToolCalls)
	}

	return nil
}

// JSONMode demonstrates structured JSON output.
func JSONMode() error {
	apiKey := os.Getenv("DEEPSEEK_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("DEEPSEEK_API_KEY environment variable is required")
	}

	client, err := deepseek.New(deepseek.Config{
		APIKey:         apiKey,
		DefaultOptions: []llm.ChatOption{llm.WithModel("deepseek-chat")},
	})
	if err != nil {
		return err
	}

	// Simple JSON object response
	resp, err := client.Chat(context.Background(), []schema.Message{
		schema.UserMessage("List 3 programming languages with their year of creation"),
	},
		llm.WithResponseFormat(schema.ResponseFormat{Type: "json_object"}),
		llm.WithMaxTokens(200),
	)
	if err != nil {
		return err
	}

	if len(resp.Choices) > 0 {
		fmt.Println("JSON Response:")
		fmt.Println(resp.Choices[0].Message.Text())
	}

	// JSON Schema for more control
	schemaJSON := json.RawMessage(`{
		"type": "object",
		"properties": {
			"name": {"type": "string"},
			"year": {"type": "integer"},
			"features": {
				"type": "array",
				"items": {"type": "string"}
			}
		},
		"required": ["name", "year", "features"],
		"additionalProperties": false
	}`)

	resp2, err := client.Chat(context.Background(), []schema.Message{
		schema.UserMessage("Tell me about Go programming language"),
	},
		llm.WithResponseFormat(schema.ResponseFormat{
			Type:       "json_schema",
			JSONSchema: schemaJSON,
		}),
		llm.WithMaxTokens(200),
	)
	if err != nil {
		return err
	}

	if len(resp2.Choices) > 0 {
		fmt.Println("\nJSON Schema Response:")
		fmt.Println(resp2.Choices[0].Message.Text())
	}

	return nil
}
