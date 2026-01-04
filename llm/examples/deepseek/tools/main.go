package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/lgc202/go-kit/llm"
	deepseek "github.com/lgc202/go-kit/llm/provider/deepseek/chat"
	"github.com/lgc202/go-kit/llm/schema"
)

// getWeather 模拟天气查询工具
func getWeather(location string) (string, error) {
	// 模拟 API 调用
	weatherData := map[string]string{
		"北京": "22°C, 晴朗",
		"上海": "25°C, 多云",
		"深圳": "28°C, 阴天",
	}

	if weather, ok := weatherData[location]; ok {
		return fmt.Sprintf("%s 的天气: %s", location, weather), nil
	}
	return fmt.Sprintf("%s 的天气: 未知", location), nil
}

func main() {
	apiKey := os.Getenv("API_KEY")
	if apiKey == "" {
		log.Fatal("API_KEY environment variable is required")
	}

	modelName := os.Getenv("MODEL")
	if modelName == "" {
		modelName = "deepseek-chat"
	}

	// 创建工具定义
	weatherTool, err := schema.NewFunctionTool(
		"get_weather",
		"获取指定地点的当前天气",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"location": map[string]any{
					"type":        "string",
					"description": "城市名称，如：北京、上海、深圳",
				},
			},
			"required": []string{"location"},
		},
	)
	if err != nil {
		log.Fatal(err)
	}

	// 创建客户端
	client, err := deepseek.New(deepseek.Config{
		BaseConfig: deepseek.BaseConfig{
			APIKey: apiKey,
		},
		DefaultOptions: []llm.ChatOption{
			llm.WithModel(modelName),
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	messages := []schema.Message{
		schema.UserMessage("北京和上海今天天气怎么样？"),
	}

	// 循环：模型可能需要多次调用工具（例如分别查询多个城市）
	const maxSteps = 8
	for step := 0; step < maxSteps; step++ {
		resp, err := client.Chat(context.Background(), messages, llm.WithTools(weatherTool))
		if err != nil {
			log.Fatal(err)
		}
		msg := resp.Choices[0].Message
		messages = append(messages, msg)

		if len(msg.ToolCalls) == 0 {
			fmt.Println("\n最终回复:")
			fmt.Println(msg.Text())
			return
		}

		fmt.Println("模型决定调用工具:")
		for _, tc := range msg.ToolCalls {
			fmt.Printf("  - 调用: %s\n", tc.Function.Name)
			fmt.Printf("    参数: %s\n", tc.Function.Arguments)

			var args struct {
				Location string `json:"location"`
			}
			if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
				log.Fatal(err)
			}

			result, err := getWeather(args.Location)
			if err != nil {
				log.Fatal(err)
			}
			messages = append(messages, schema.ToolResultMessage(tc.ID, result))
		}
	}

	log.Fatalf("exceeded max tool-call steps (%d)", maxSteps)
}
