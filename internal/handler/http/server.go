// internal/handler/http/server.go

package http

import (
	"context"
	"file-transfer-api/internal/domain"
	"file-transfer-api/internal/pkg/config" // 💡 configをここで使う
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"
)

// Server は HTTP サーバーのライフサイクルを管理する構造体
type Server struct {
	echo *echo.Echo
	port string
}

// NewServer はコンストラクタ。必要な依存関係（UseCase, Repo, Config）をすべて注入する
func NewServer(interactor domain.FileUseCase, metadataRepo domain.MetadataRepository, cfg *config.Config) *Server {
	e := NewRouter(interactor, metadataRepo) // 既存のルーター生成ロジックをそのまま利用

	return &Server{
		echo: e,
		port: cfg.HTTPPort, // 💡 パース済みの安全な設定値を使用
	}
}

// Start はサーバーを同期起動する（ブロックする）
func (s *Server) Start() error {
	slog.Info("📡 Starting HTTP server", "port", s.port)

	// 🚀 Echo スタイルの起動
	if err := s.echo.Start(":" + s.port); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

// Shutdown は安全な停止（Graceful Shutdown）を提供する
func (s *Server) Shutdown(ctx context.Context) error {
	slog.Info("🛑 Shutting down HTTP server gracefully...")
	return s.echo.Shutdown(ctx)
}
