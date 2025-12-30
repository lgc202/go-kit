# llm

用于与大模型交互的工具包。

目录结构（简要）：

- `schema/`: 统一的消息/工具/响应结构
- `provider/`: 各厂商实现（OpenAI-compatible 等）
- `internal/openai_compat/`: OpenAI-compatible 兼容层（供 provider 复用）
- `examples/`: 使用示例

补充：

- 多模态消息：使用 `schema.Message.Content` 搭配 `schema.TextPart` / `schema.ImageURLPart` / `schema.BinaryPart`
- OpenAI-compatible 扩展：使用 `llm.WithExtraHeaders` / `llm.WithExtraFields` 注入厂商差异字段
- 流式回调：使用 `llm.Wrap(model).Chat(..., llm.WithStreamingFunc(...))` 会自动走 `ChatStream` 并边收边回调
