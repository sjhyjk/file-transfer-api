// internal/handler/rest/file_handler_http.go

package rest

import (
	"file-transfer-api/internal/domain"
	"file-transfer-api/internal/handler"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

// 1. 構造体の定義（ServerInterfaceを実装する）
// ServerInterface は oapi-codegen が生成するインターフェース
// これを実装することで、スキーマ通りのハンドラーであることを保証します
type HTTPFileHandler struct {
	interactor domain.FileUseCase
}

func NewHTTPFileHandler(interactor domain.FileUseCase) *HTTPFileHandler {
	return &HTTPFileHandler{interactor: interactor}
}

// 2. メタデータ一覧取得の実装
// 🚀 自動生成されたインターフェース（ListFiles）を実装する形になります
// GetFiles は OpenAPI の operationId: getFiles に対応して自動で呼ばれます
// tags や params は定義済み型として渡されるので、strconv.Atoi は不要になります！
func (h *HTTPFileHandler) ListFiles(ctx echo.Context, params handler.ListFilesParams) error {
	// params.Limit には、すでに int 型で値が入っています。
	// もし limit に文字列が送られてきたら、このメソッドが呼ばれる前に
	// ライブラリ側で 400 Bad Request を返してくれます。

	// 1. Contextの取得（TraceID入り）
	rCtx := ctx.Request().Context()

	// 2. パラメータの整理（ポインタ解除）
	// 🚀 params.Tags は自動的に []string になっています！
	var tags []string
	if params.Tags != nil {
		tags = *params.Tags // ポインタから実体を取得
		slog.InfoContext(ctx.Request().Context(), "Filtering by tags", "tags", tags)
	}

	limit := 20
	if params.Limit != nil {
		limit = *params.Limit
	}

	// 3. ロジック実行
	// 🚀 注目：params.Limit や params.Tags は自動で型変換済み
	// Usecaseの呼び出し
	domainFiles, err := h.interactor.FetchMetadataList(rCtx, tags, limit, 0)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, map[string]string{"error": "Internal Server Error"})
	}

	// 🚀 ポイント：domain 層の型を handler 層（OpenAPI生成型）へ詰め替える
	// これにより、API仕様外の内部データ（UpdatedAtなど）を隠蔽する役割も果たします。
	resp := make([]handler.FileMetadata, len(domainFiles))
	for i, f := range domainFiles {
		// 1. IDの変換 (int64 -> UUIDポインタ)
		// OpenAPIがUUID型を期待しているので、形式を合わせる必要があります
		idStr := fmt.Sprintf("%08x-0000-0000-0000-%012x", 0, f.ID) // 暫定的なUUID化
		u, _ := uuid.Parse(idStr)                                  // 標準的な google/uuid でパース
		parsedUUID := openapi_types.UUID(u)

		// 2. 基本型のポインタ詰め替え
		fileName := f.FileName
		fileSize := int(f.FileSize)
		status := handler.FileMetadataStatus(f.Status) // ドメインのStatusをAPIのEnum型にキャスト

		resp[i] = handler.FileMetadata{
			Id:     &parsedUUID,
			Name:   &fileName, // FileName ではなく Name (OpenAPIの定義通り)
			Size:   &fileSize,
			Status: &status,
			Tags:   &f.Tags,
		}
	}

	// 4. レスポンス（ヘッダー設定も自動）
	return ctx.JSON(http.StatusOK, resp)
}

// 3. 他のメソッド（GetHealth, UploadFile）も同様に「器」だけ作ります
func (h *HTTPFileHandler) GetHealth(ctx echo.Context) error {
	// ベンチマーク結果などをここで返す
	return ctx.String(http.StatusOK, "OK")
}

// POST /upload の実装例
// UploadFile は multipart/form-data を解析し、ファイルをアップロードします
func (h *HTTPFileHandler) UploadFile(ctx echo.Context) error {
	rCtx := ctx.Request().Context()

	// 1. ファイルの取得
	// OpenAPIで定義した "file" というキーでファイルを取り出す
	fileHeader, err := ctx.FormFile("file")
	if err != nil {
		slog.ErrorContext(rCtx, "Failed to get file from form", "error", err)
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": "file is required"})
	}

	// 2. その他のパラメータ取得
	tenantID := ctx.FormValue("tenant_id")
	// tagsは複数送られてくる可能性があるため FormParams を使う
	tags := ctx.Request().Form["tags"]

	// 🚀 ポイント：tenantID と tags をログに出力することで「未使用エラー」を回避しつつ、
	// 実務で重要な「誰が何をしようとしているか」の証跡を残します。
	slog.InfoContext(rCtx, "Processing upload request",
		"tenant_id", tenantID,
		"tags", tags,
		"filename", fileHeader.Filename,
	)

	// 3. ファイルを読み込み可能な状態にする
	src, err := fileHeader.Open()
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to open file"})
	}

	// 🚀 修正：defer で Close のエラーを適切にハンドルする
	defer func() {
		if closeErr := src.Close(); closeErr != nil {
			slog.WarnContext(rCtx, "Failed to close uploaded file", "error", closeErr)
		}
	}()

	// 4. Domainモデルへの変換
	// interactor が期待する domain.File 型を作る
	// domain.NewFile(name, size, reader) のような関数があればそれを使います
	f, err := domain.NewFile(fileHeader.Filename, fileHeader.Size, src)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	// TODO: domain.File 構造体に Tags フィールドを追加したら以下のコメントを外す
	// f.Tags = tags
	// f.TenantID = tenantID

	// 5. ロジック実行（並行アップロードではなく、単発アップロードを呼ぶ）
	// もし UploadMultipleParallel しか無い場合はスライスに入れて渡します
	err = h.interactor.UploadMultipleParallel(rCtx, []*domain.File{f})
	if err != nil {
		slog.ErrorContext(rCtx, "Upload failed", "file", f.Name, "error", err)
		return ctx.JSON(http.StatusInternalServerError, map[string]string{"error": "upload failed"})
	}

	// 6. レスポンス（OpenAPIのスキーマに合わせる）
	return ctx.JSON(http.StatusCreated, map[string]string{
		"status":  "success",
		"message": "File uploaded successfully",
	})
}
