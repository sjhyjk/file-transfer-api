package handler

import (
	"encoding/json"
	"file-transfer-api/internal/usecase"
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
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	// 1. クエリパラメータの解析 (limit, offset)
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	// 2. Usecase の呼び出し
	files, err := h.interactor.FetchMetadataList(ctx, limit, offset)
	if err != nil {
		http.Error(w, "Failed to fetch files: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 3. レスポンスの返却
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(files); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}
