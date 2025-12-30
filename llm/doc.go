// Package llm provides a provider-agnostic chat SDK.
//
// Design goals:
//   - Stable domain model: callers build requests using canonical types (ChatRequest, Message, ToolDefinition).
//   - Explicit streaming: providers emit StreamEvent values (part/tool deltas, usage, done) and callers can
//     reconstruct final responses using Accumulator or DrainStream.
//   - Controlled escape hatches: provider-specific fields can be passed via ChatRequest.Extra, and
//     request-scoped headers via ChatRequest.Transport.
//
// Provider implementations live under llm/providers and are responsible for mapping between the
// canonical model and each provider's wire format.
package llm
