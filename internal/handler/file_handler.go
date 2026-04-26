package handler

import (
	"encoding/json"
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
	ctx := r.Context()

	// 🚀 1. 開始ログ：どんな条件でリクエストが来たか記録
	slog.InfoContext(ctx, "Handling list files request",
		"method", r.Method,
		"path", r.URL.Path,
		"tags", r.URL.Query()["tags"],
	)

	// 2. メソッドバリデーション
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 3. クエリパラメータの解析（ページネーション & 検索タグ）
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	// 🚀 タグ検索用のパラメータ取得
	// ?tags=golang,aws のようなカンマ区切り、または ?tags=golang&tags=aws の複数指定を想定
	tags := r.URL.Query()["tags"]
	// ※ tags := r.URL.Query().Get("tags") ではなく []string で取得できるこの書き方が便利です

	// 4. Usecase の呼び出し（tags を追加）
	files, err := h.interactor.FetchMetadataList(ctx, tags, limit, offset)
	if err != nil {
		// 🚀 5. エラーログ：何が原因で失敗したか属性付きで記録
		slog.ErrorContext(ctx, "Failed to fetch metadata list",
			"error", err,
			"tags", tags,
		)
		http.Error(w, "Failed to fetch files: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 6. レスポンスの返却
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(files); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}
