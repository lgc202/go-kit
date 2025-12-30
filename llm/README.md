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
- `llm/providers/kimi`: Kimi (Moonshot) OpenAI-compatible provider wrapper
- `llm/providers/ollama`: Ollama OpenAI-compatible provider wrapper

## Quick start

```go
provider, _ := openai.New(os.Getenv("OPENAI_API_KEY"), openai.WithDefaultModel("gpt-4o-mini"))
client := llm.New(provider, llm.WithTemperature(0.7))

resp, _ := client.Chat(ctx, []llm.Message{llm.User("Hello")})
fmt.Println(resp.FirstText())
```

## Streaming

```go
stream, _ := client.ChatStream(ctx, []llm.Message{llm.User("Explain SSE.")})
defer stream.Close()

for {
  ev, err := stream.Recv()
  if err != nil { break }
  if ev.Kind == llm.StreamEventPartDelta && ev.PartDelta != nil && ev.PartDelta.Type == llm.ContentPartText {
    fmt.Print(ev.PartDelta.TextDelta)
  }
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
resp, _ := client.Chat(ctx, msgs,
  llm.WithModel("..."),
  llm.WithTools(tools...),
  llm.WithToolChoice(tc),
)
```

When the assistant returns `Message.ToolCalls`, execute them and send results back using a `RoleTool` message with `ToolCallID`.

## Thinking / Reasoning (DeepSeek)

Some providers return a separate reasoning channel (e.g. DeepSeek `reasoning_content` / `thinking`).

- Non-streaming: mapped to `llm.Message.Parts` as `llm.ContentPartReasoning` (use `msg.Reasoning()` helper)
- Streaming: emitted as `llm.StreamEventPartDelta` with `PartDelta.Type == llm.ContentPartReasoning`

DeepSeek also supports controlling thinking in the request:

```go
provider, _ := deepseek.New(os.Getenv("DEEPSEEK_API_KEY"),
  deepseek.WithDefaultModel("deepseek-chat"),
)

// client-level default (applies to all requests)
client := llm.New(provider, deepseek.WithThinkingDisabled())

// Or override per-request:
resp, _ := client.Chat(ctx, msgs, deepseek.WithThinkingEnabled())
```

## Providers

- OpenAI: `openai.New(key, openai.WithDefaultModel("...") )`
- DeepSeek: `deepseek.New(key, deepseek.WithDefaultModel("deepseek-chat"))`
- Qwen (DashScope compatible-mode): `qwen.New(key, qwen.WithDefaultModel("qwen-plus"))`
- Kimi (Moonshot): `kimi.New(key, kimi.WithDefaultModel("moonshot-v1-8k"))`
- Ollama (local): `ollama.New("", ollama.WithDefaultModel("llama3"))`

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

## Raw JSON (debug)

By default, providers avoid copying raw JSON payloads for performance.

- Non-streaming raw response: enable via `openai_compat.WithIncludeRawResponse(true)` (populates `llm.ChatResponse.RawJSON`).
- Streaming raw chunks: enable via `openai_compat.WithIncludeRawStreamEvents(true)` (populates `llm.StreamEvent.RawJSON`).
