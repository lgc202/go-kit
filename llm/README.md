# LLM SDK

Provider-agnostic Go `llm.Client` with:

- Canonical domain types (`llm.ChatRequest`, `llm.ChatResponse`, `llm.StreamEvent`)
- Streaming via SSE (`llm.Stream`)
- Tool/function calling (`llm.ToolDefinition`, `llm.ToolCall`)
- Enterprise-friendly errors (`llm.LLMError`) and retry-capable HTTP transport

## Packages

- `llm`: domain model + `llm.Client`
- `llm/providers/openai`: OpenAI provider (Chat Completions)
- `llm/providers/openai_compat`: OpenAI-compatible provider base (patch via options/hooks)
- `llm/providers/deepseek`: example OpenAI-compatible provider wrapper
- `llm/providers/qwen`: Qwen (DashScope) OpenAI-compatible provider wrapper

## Quick start

```go
provider, _ := openai.New(os.Getenv("OPENAI_API_KEY"), openai.WithDefaultModel("gpt-4o-mini"))
client := llm.New(provider)

req := llm.NewChatRequest("", llm.Message{Role: llm.RoleUser, Content: "Hello"}).
  With(llm.WithTemperature(0.7))

resp, _ := client.Chat(ctx, req)
fmt.Println(resp.FirstText())
```

## Streaming

```go
stream, _ := client.ChatStream(ctx, llm.NewChatRequest("", llm.Message{Role: llm.RoleUser, Content: "Explain SSE."}))
defer stream.Close()

for {
  ev, err := stream.Recv()
  if err != nil { break }
  if ev.Kind == llm.StreamEventTextDelta { fmt.Print(ev.TextDelta) }
  if ev.Done() { break }
}
```

If you want a complete final response from a stream:

```go
resp, err := llm.DrainStream(stream)
```

## Tool calling

Define tools:

```go
tools := []llm.ToolDefinition{
  {
    Name: "get_weather",
    Description: "Get weather by location",
    InputSchema: json.RawMessage(`{"type":"object","properties":{"location":{"type":"string"}},"required":["location"]}`),
  },
}
```

Pass them in the request:

```go
tc := llm.AutoToolChoice()
req := llm.NewChatRequest("...", msgs...).With(
  llm.WithTools(tools...),
  llm.WithToolChoice(tc),
)
```

When the assistant returns `Message.ToolCalls`, execute them and send results back using a `RoleTool` message with `ToolCallID`.

## Thinking / Reasoning (DeepSeek)

Some providers return a separate reasoning channel (e.g. DeepSeek `reasoning_content` / `thinking`).

- Non-streaming: mapped to `llm.Message.Reasoning`
- Streaming: emitted as `llm.StreamEventReasoningDelta` and accumulated into `llm.Message.Reasoning`

DeepSeek also supports controlling thinking in the request:

```go
provider, _ := deepseek.New(os.Getenv("DEEPSEEK_API_KEY"),
  deepseek.WithDefaultModel("deepseek-chat"),
  // provider-level default (applies to all requests)
  deepseek.WithDefaultRequest(deepseek.WithThinkingDisabled()),
)

// Or override per-request:
req := llm.NewChatRequest("", msgs...).With(
  deepseek.WithThinkingEnabled(),
)
```

## Providers

- OpenAI: `openai.New(key, openai.WithDefaultModel("...") )`
- DeepSeek: `deepseek.New(key, deepseek.WithDefaultModel("deepseek-chat"))`
- Qwen (DashScope compatible-mode): `qwen.New(key, qwen.WithDefaultModel("qwen-plus"))`

## Request fields

`llm.ChatRequest` supports common OpenAI-compatible parameters, including:

- `Temperature`, `TopP`, `MaxTokens`, `Seed`
- `PresencePenalty`, `FrequencyPenalty`, `Stop`
- `ResponseFormat` (`text`, `json_object`, `json_schema`)
- `LogProbs`, `TopLogProbs`
- `StreamOptions` (e.g. include usage)

Provider-specific knobs can be passed via `ChatRequest.Extra`.
To send an explicit JSON `null` for a field, set `Extra["field"] = nil`.

## Handling provider differences

Use `openai_compat.WithHooks` to patch headers or requests (for vendors that are “OpenAI-compatible” but not identical).

## Error handling

Provider/HTTP failures are returned as `*llm.LLMError` (when possible):

```go
if e, ok := llm.AsLLMError(err); ok {
  fmt.Println(e.Kind, e.HTTPStatus, e.Retryable)
}
```
