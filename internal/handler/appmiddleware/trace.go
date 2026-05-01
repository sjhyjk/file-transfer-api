package appmiddleware

import (
	"file-transfer-api/internal/pkg/logger"
	"net/http"
)

// TraceMiddleware は、すべてのHTTPリクエストに一意の Trace ID を付与します。
// TraceMiddleware は、標準の http.Handler を受け取る形に書き換えます
func TraceMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// クライアントから ID が送られてきている場合はそれを尊重
		traceID := r.Header.Get("X-Trace-Id")

		// ID入りの context を作成
		ctx := logger.WithTraceID(r.Context(), traceID)

		// ID入りの context をセットして次に渡す
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
