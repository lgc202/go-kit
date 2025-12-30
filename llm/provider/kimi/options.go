package kimi

// Kimi (Moonshot AI) 基本上是 OpenAI 兼容的 API
// 所有标准参数都通过 llm 包的 RequestOption 设置
// 无需额外的厂商特定参数

// 如需设置请求参数，使用 llm 包提供的选项函数，例如：
//
//	llm.WithModel("moonshot-v1-8k")
//	llm.WithTemperature(0.7)
//	llm.WithMaxCompletionTokens(4096)
