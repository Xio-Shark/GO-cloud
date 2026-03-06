package traceutil

import (
	"context"
	"crypto/rand"
	"encoding/hex"
)

type contextKey string

const traceIDKey contextKey = "trace_id"

func NewTraceID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "trace-id-fallback"
	}
	return hex.EncodeToString(buf)
}

func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDKey, traceID)
}

func FromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	traceID, ok := ctx.Value(traceIDKey).(string)
	if !ok {
		return ""
	}
	return traceID
}
