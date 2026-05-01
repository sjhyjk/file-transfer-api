package handler

import (
	"file-transfer-api/internal/domain"
	"file-transfer-api/internal/usecase"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"
)

// 1. 構造体の定義（ServerInterfaceを実装する）
// ServerInterface は oapi-codegen が生成するインターフェース
// これを実装することで、スキーマ通りのハンドラーであることを保証します
type FileHandler struct {
	interactor *usecase.FileInteractor
}

func NewFileHandler(interactor *usecase.FileInteractor) *FileHandler {
	return &FileHandler{interactor: interactor}
}

// 2. メタデータ一覧取得の実装
// 🚀 自動生成されたインターフェース（ListFiles）を実装する形になります
// GetFiles は OpenAPI の operationId: getFiles に対応して自動で呼ばれます
// tags や params は定義済み型として渡されるので、strconv.Atoi は不要になります！
func (h *FileHandler) ListFiles(ctx echo.Context, params ListFilesParams) error {
	// params.Limit には、すでに int 型で値が入っています。
	// もし limit に文字列が送られてきたら、このメソッドが呼ばれる前に
	// ライブラリ側で 400 Bad Request を返してくれます。

	// 1. Contextの取得（TraceID入り）
	rCtx := ctx.Request().Context()

	// 2. パラメータの整理（ポインタ解除）
	// 🚀 params.Tags は自動的に []string になっています！
	var tags []string
	if params.Tags != nil {
		tags = *params.Tags
		slog.InfoContext(ctx.Request().Context(), "Filtering by tags", "tags", tags)
	}

	limit := 20
	if params.Limit != nil {
		limit = *params.Limit
	}

	// 3. ロジック実行
	// 🚀 注目：params.Limit や params.Tags は自動で型変換済み
	// Usecaseの呼び出し
	files, err := h.interactor.FetchMetadataList(ctx.Request().Context(), tags, limit, 0)
	if err != nil {
		// エラーハンドリング
		slog.ErrorContext(rCtx, "Failed to fetch metadata list", "error", err)
		return ctx.JSON(http.StatusInternalServerError, map[string]string{"error": "Internal Server Error"})
	}

	// 4. レスポンス（ヘッダー設定も自動）
	// 🚀 json.Encode を手書きする必要がなくなり、1行で終わります
	return ctx.JSON(http.StatusOK, files)
}

// 3. 他のメソッド（GetHealth, UploadFile）も同様に「器」だけ作ります
func (h *FileHandler) GetHealth(ctx echo.Context) error {
	// ベンチマーク結果などをここで返す
	return ctx.String(http.StatusOK, "OK")
}

// POST /upload の実装例
// UploadFile は multipart/form-data を解析し、ファイルをアップロードします
func (h *FileHandler) UploadFile(ctx echo.Context) error {
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
	defer src.Close()

	// 4. Domainモデルへの変換
	// interactor が期待する domain.File 型を作る
	// domain.NewFile(name, size, reader) のような関数があればそれを使います
	f := domain.NewFile(fileHeader.Filename, fileHeader.Size, src)

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
