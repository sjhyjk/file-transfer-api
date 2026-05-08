// internal/handler/rest/router.go

package rest

import (
	"file-transfer-api/internal/domain"
	"file-transfer-api/internal/handler/rest/appmiddleware"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// NewRouter は設定済みの Echo インスタンスを返します
func NewRouter(interactor domain.FileUseCase, metadataRepo domain.MetadataRepository) *echo.Echo {
	e := echo.New()

	// ミドルウェア一括設定
	e.Use(echo.WrapMiddleware(appmiddleware.TraceMiddleware))
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogStatus:   true,
		LogURI:      true,
		LogMethod:   true,
		HandleError: true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			// slog で出力
			return nil // 必要に応じてここで slog.Info を呼ぶ設定も可能です
		},
	}))
	e.Use(middleware.Recover()) // パニック時に落ちないように

	// ハンドラー登録
	// これにより YAML で定義した /files, /upload 等が紐付きます
	h := NewHTTPFileHandler(interactor)
	RegisterHandlers(e, h)

	// ヘルスチェック (OpenAPI外の自由なルート)
	e.GET("/health", func(c echo.Context) error {
		dbStatus := "OK"
		if metadataRepo == nil {
			dbStatus = "NG"
		}
		// improvement は一旦 0 または起動パラメータから渡すようにします
		// interactor 経由で DB 疎通を見るのがより正確です
		return c.String(http.StatusOK, fmt.Sprintf("✅ Running! DB: %s", dbStatus))
	})

	return e
}

// StartServer はポートの取得から HTTP サーバーの起動までを担当します
func StartServer(interactor domain.FileUseCase, metadataRepo domain.MetadataRepository) {
	e := NewRouter(interactor, metadataRepo)

	// ポート取得ロジックをここに隠蔽
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // ローカル実行時のデフォルト
	}

	slog.Info("📡 Starting HTTP server", "port", port)

	// 🚀 Echo スタイルの起動
	if err := e.Start(":" + port); err != nil && err != http.ErrServerClosed {
		slog.Error("❌ HTTP server failed to start", "error", err)
		os.Exit(1)
	}
}
