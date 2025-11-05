package constants

// Context keys - used across multiple files to prevent typos
const (
	CtxKeyToken     = "token"
	CtxKeyUserID    = "user_id"
	CtxKeyRequestID = "request_id"
)

// Attribute keys for tracing and logging consistency
const (
	AttrKeyUserID  = "user_id"
	AttrKeyEmail   = "email"
	AttrKeyTraceID = "trace_id"
	AttrKeySpanID  = "span_id"
)
