// internal/handler/rest/appmiddleware/trace.go

package appmiddleware

import (
	"context"
	"file-transfer-api/internal/pkg/logger"
	"net/http"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// TraceMiddleware は、すべてのHTTPリクエストに一意の Trace ID を付与します。
// TraceMiddleware は、標準の http.Handler を受け取る形に書き換えます
func TraceMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// クライアントから ID が送られてきている場合はそれを尊重
		traceID := r.Header.Get("x-trace-id")

		// ID入りの context を作成
		ctx := logger.WithTraceID(r.Context(), traceID)

		// 🚀 生成された or 確定した本物の Trace ID をレスポンスヘッダーにも仕込む
		actualTraceID := logger.FromContext(ctx)
		w.Header().Set("x-trace-id", actualTraceID)

		// ID入りの context をセットして次に渡す
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// TraceStreamServerInterceptor は、ストリーミングRPCに対して Trace ID を付与するミドルウェアです。
func TraceStreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx := ss.Context()
		var traceID string

		// gRPCのメタデータから X-Trace-Id を探す
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if ids := md.Get("x-trace-id"); len(ids) > 0 {
				traceID = ids[0]
			}
		}

		// 既存の共通ロジックを使って context に trace_id を付与
		// (id が空なら内部で uuid.New().String() が走る)
		newCtx := logger.WithTraceID(ctx, traceID)

		// 変更した context を持つ新しいストリームラッパーを作成してハンドラーに渡す
		wrapped := &wrappedStream{ServerStream: ss, ctx: newCtx}
		return handler(srv, wrapped)
	}
}

// wrappedStream は context を差し替えるためのヘルパー構造体です
type wrappedStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedStream) Context() context.Context {
	return w.ctx
}
