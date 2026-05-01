package logger

import (
	"context"

	"github.com/google/uuid"
)

// context の Key は衝突を避けるために独自型で定義
type ctxKey struct{}

var traceIDKey = ctxKey{}

// WithTraceID は context に Trace ID を埋め込んだ新しい context を返します
func WithTraceID(ctx context.Context, id string) context.Context {
	if id == "" {
		id = uuid.New().String() // 指定がなければ新規生成
	}
	return context.WithValue(ctx, traceIDKey, id)
}

// FromContext は context から Trace ID を取り出します
func FromContext(ctx context.Context) string {
	if id, ok := ctx.Value(traceIDKey).(string); ok {
		return id
	}
	return ""
}
