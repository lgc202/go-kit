package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/lgc202/go-kit/llm"
	ollama "github.com/lgc202/go-kit/llm/provider/ollama/chat"
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
	modelName := os.Getenv("MODEL")
	if modelName == "" {
		modelName = "llama3.2"
	}

	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:11434/v1"
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
	client, err := ollama.New(ollama.Config{
		BaseConfig: ollama.BaseConfig{
			BaseURL: baseURL,
		},
		DefaultOptions: []llm.ChatOption{
			llm.WithModel(modelName),
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	// 第一轮：用户询问天气
	messages := []schema.Message{
		schema.UserMessage("北京和上海今天天气怎么样？"),
	}

	resp, err := client.Chat(context.Background(), messages,
		llm.WithTools(weatherTool),
	)
	if err != nil {
		log.Fatal(err)
	}

	msg := resp.Choices[0].Message

	// 检查是否需要调用工具
	if len(msg.ToolCalls) > 0 {
		fmt.Println("模型决定调用工具:")

		// 添加助手的回复到历史
		messages = append(messages, msg)

		// 执行每个工具调用
		for _, tc := range msg.ToolCalls {
			fmt.Printf("  - 调用: %s\n", tc.Function.Name)
			fmt.Printf("    参数: %s\n", tc.Function.Arguments)

			// 解析参数
			var args struct {
				Location string `json:"location"`
			}
			if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
				log.Fatal(err)
			}

			// 调用工具
			result, err := getWeather(args.Location)
			if err != nil {
				log.Fatal(err)
			}

			// 添加工具结果到历史
			messages = append(messages, schema.ToolResultMessage(tc.ID, result))
		}

		// 第二轮：发送工具结果回模型
		resp2, err := client.Chat(context.Background(), messages)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println("\n最终回复:")
		fmt.Println(resp2.Choices[0].Message.Text())
	} else {
		fmt.Println("模型回复:", msg.Text())
	}
}
