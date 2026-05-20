package voicelog

import (
	"context"
	"os"

	"github.com/rs/zerolog"
)

var Logger = zerolog.New(os.Stdout).With().Timestamp().Logger()

type contextKey string

const (
	callIDKey    contextKey = "call_id"
	sessionIDKey contextKey = "session_id"
	tenantIDKey  contextKey = "tenant_id"
)

func WithContext(ctx context.Context, callID, sessionID, tenantID string) context.Context {
	ctx = context.WithValue(ctx, callIDKey, callID)
	ctx = context.WithValue(ctx, sessionIDKey, sessionID)
	ctx = context.WithValue(ctx, tenantIDKey, tenantID)
	return ctx
}

func FromContext(ctx context.Context) zerolog.Logger {
	l := Logger
	if v, ok := ctx.Value(callIDKey).(string); ok {
		l = l.With().Str("call_id", v).Logger()
	}
	if v, ok := ctx.Value(sessionIDKey).(string); ok {
		l = l.With().Str("session_id", v).Logger()
	}
	if v, ok := ctx.Value(tenantIDKey).(string); ok {
		l = l.With().Str("tenant_id", v).Logger()
	}
	return l
}
