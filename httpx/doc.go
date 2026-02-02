// Package httpx provides an "enterprise-grade" HTTP client wrapper:
// - safe, reusable transports with sane defaults
// - request building with base URL + default headers
// - retry with exponential backoff + jitter (idempotent methods by default)
// - error type carrying status, request id, retry-after and limited body
// - hook points for logging/metrics/tracing without hard dependencies
package httpx
