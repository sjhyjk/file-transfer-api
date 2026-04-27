package handler

import (
	"file-transfer-api/internal/pkg/requestid"
	"net/http"
)

// TraceMiddleware は、すべてのHTTPリクエストに一意の Trace ID を付与します。
func TraceMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// クライアントから ID が送られてきている場合はそれを尊重し、
		// なければ新規発行する（requestid.WithTraceID の仕様）
		traceID := r.Header.Get("X-Trace-Id")
		ctx := requestid.WithTraceID(r.Context(), traceID)

		// ID入りの context をリクエストにセットして次に渡す
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}
