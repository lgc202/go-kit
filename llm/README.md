# llm

用于与大模型交互的工具包。

目录结构（简要）：

- `schema/`: 统一的消息/工具/响应结构
- `provider/`: 各厂商实现（OpenAI-compatible 等）
- `provider/*/chat`: Chat client（按 endpoint 拆包，如 `provider/openai/chat`）
- `provider/*/embeddings`: Embeddings client（按 endpoint 拆包，如 `provider/openai/embeddings`）
- `internal/openai_compat/transport`: OpenAI-compatible 传输层复用（HTTP/headers/error）
- `internal/openai_compat/chat`: OpenAI-compatible ChatCompletions 兼容实现
- `internal/openai_compat/embeddings`: OpenAI-compatible Embeddings 兼容实现
- `examples/`: 使用示例

补充：

- 多模态消息：使用 `schema.Message.Content` 搭配 `schema.TextPart` / `schema.ImageURLPart` / `schema.BinaryPart`
- OpenAI-compatible 扩展：使用 `llm.WithExtraHeaders` / `llm.WithExtraFields` 注入厂商差异字段
- `ExtraFields`：默认不允许覆盖内置字段（如 `model`），如需覆盖用 `llm.WithAllowExtraFieldOverride(true)`
- 流式调用：直接使用 `ChatStream` 接口逐块读取
