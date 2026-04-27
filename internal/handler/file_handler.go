package handler

import (
	"encoding/json"
	"file-transfer-api/internal/pkg/requestid"
	"file-transfer-api/internal/usecase"
	"log/slog"
	"net/http"
	"strconv"
)

type FileHandler struct {
	interactor *usecase.FileInteractor
}

func NewFileHandler(interactor *usecase.FileInteractor) *FileHandler {
	return &FileHandler{interactor: interactor}
}

// HandleListFiles は GET /files エエンドポイントを処理します
func (h *FileHandler) HandleListFiles(w http.ResponseWriter, r *http.Request) {
	// 🚀 1. Trace ID を生成して context に注入
	// クライアントから X-Trace-Id ヘッダーがあればそれを使う、なければ新規発行する実装がプロっぽいです
	traceID := r.Header.Get("X-Trace-Id")
	ctx := requestid.WithTraceID(r.Context(), traceID)

	// 🚀 2. 開始ログ（以降、ctx を使用する）
	slog.InfoContext(ctx, "Handling list files request",
		"method", r.Method,
		"path", r.URL.Path,
		"tags", r.URL.Query()["tags"],
	)

	// 3. メソッドバリデーション
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 4. クエリパラメータの解析（ページネーション & 検索タグ）
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	// 🚀 タグ検索用のパラメータ取得
	// ?tags=golang,aws のようなカンマ区切り、または ?tags=golang&tags=aws の複数指定を想定
	tags := r.URL.Query()["tags"]
	// ※ tags := r.URL.Query().Get("tags") ではなく []string で取得できるこの書き方が便利です

	// 🚀 5. Usecase の呼び出し（注入した ID 入りの ctx を渡す）
	files, err := h.interactor.FetchMetadataList(ctx, tags, limit, offset)
	if err != nil {
		// 🚀 6. エラーログ：何が原因で失敗したか属性付きで記録
		slog.ErrorContext(ctx, "Failed to fetch metadata list",
			"error", err,
			"tags", tags,
		)
		http.Error(w, "Failed to fetch files: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 7. レスポンスの返却
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(files); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}
