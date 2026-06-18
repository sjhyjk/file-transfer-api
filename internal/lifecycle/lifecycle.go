// internal/lifecycle/lifecycle.go

package lifecycle

import (
	"context"
	"file-transfer-api/internal/handler/grpc"
	"file-transfer-api/internal/handler/http"
	"file-transfer-api/internal/pkg/config"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type ApplicationRunner struct {
	httpServer *http.Server
	grpcServer *grpc.Server
	cfg        *config.Config
}

func NewApplicationRunner(h *http.Server, g *grpc.Server, cfg *config.Config) *ApplicationRunner {
	return &ApplicationRunner{
		httpServer: h,
		grpcServer: g,
		cfg:        cfg,
	}
}

// ---------------------------------------------------------
// 🛑 Graceful Shutdown (安全な停止) 制御マネジメント
// ---------------------------------------------------------
// Run はアプリケーションの起動と Graceful Shutdown のライフサイクルを統括します
func (runner *ApplicationRunner) Run() {
	slog.Info("🚀 Launching Application Protocols", "mode", runner.cfg.ServerMode)

	// 1. 各サーバーをバックグラウンドで非同期起動
	serverErrors := runner.startServers()

	// 2. 🛑 OSシグナルおよびサーバー異常停止の監視
	// OSからのCtrl+C(SIGINT)や、コンテナ停止シグナル(SIGTERM)を監視するチャネル
	// OSからの停止シグナルか、サーバー自身の起動エラーをどちらか早い方で検知する
	shutdownSignals := make(chan os.Signal, 1)
	signal.Notify(shutdownSignals, os.Interrupt, syscall.SIGTERM)

	// 何かが起きるまでメインスレッドをここで待機させる
	select {
	case err := <-serverErrors:
		// サーバーがポート競合などのエラーで自爆した場合
		slog.Error("❌ Server forced to shutdown due to startup error", "error", err)
		os.Exit(1)
	case sig := <-shutdownSignals:
		// ユーザーやクローズ命令によってシグナルが届いた場合
		slog.Info("🛑 Shutdown signal received", "signal", sig.String())

		// リクエストの処理待ちリミットを5秒に設定
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// 双方のサーバーを安全にクローズ
		runner.grpcServer.GracefulStop()
		if err := runner.httpServer.Shutdown(ctx); err != nil {
			slog.Error("❌ Error during HTTP graceful shutdown", "error", err)
		}
		slog.Info("✨ All servers stopped cleanly. Cleaning up infrastructure connections...")
		// 💡 このあと、関数を抜けることでmainで宣言した defer クリーナップ群が順に実行されます
	}
}

// startServers は設定モードに応じて、各サーバーを別ゴルーチンで起動します（責務の分離）
func (runner *ApplicationRunner) startServers() <-chan error {
	serverErrors := make(chan error, 2)

	// モードに応じた go routine 起動ロジック
	switch runner.cfg.ServerMode {
	case "GRPC":
		slog.Info("Starting application in gRPC mode", "port", runner.cfg.GRPCPort)
		go func() { serverErrors <- runner.grpcServer.Start() }()
	case "HTTP":
		slog.Info("Starting application in HTTP mode", "port", runner.cfg.HTTPPort)
		go func() { serverErrors <- runner.httpServer.Start() }()
	case "BOTH":
		slog.Info("🚀 Starting application in HYBRID mode (HTTP & gRPC)",
			"http_port", runner.cfg.HTTPPort, "grpc_port", runner.cfg.GRPCPort)
		// メインスレッドが落ちないように、gRPCを裏で、HTTPを表で安全に制御
		// (本格的には context と sync.WaitGroup や golang.org/x/sync/errgroup を使うとより堅牢です)
		go func() { serverErrors <- runner.grpcServer.Start() }()
		go func() { serverErrors <- runner.httpServer.Start() }()
	default:
		// どちらも指定がなければ、安全のためにエラーにする
		slog.Error("Unknown SERVER_MODE", "mode", runner.cfg.ServerMode)
		os.Exit(1)
	}
	return serverErrors
}
