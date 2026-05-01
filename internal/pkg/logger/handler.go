package logger

import (
	"context"
	"log/slog"
)

type TraceHandler struct {
	slog.Handler
}

// Handle はログ出力のたびに呼ばれるメソッド
func (h *TraceHandler) Handle(ctx context.Context, r slog.Record) error {
	if id := FromContext(ctx); id != "" {
		// ログレコードに "trace_id" フィールドを追加
		r.AddAttrs(slog.String("trace_id", id))
	}
	return h.Handler.Handle(ctx, r)
}
