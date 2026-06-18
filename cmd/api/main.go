// cmd/api/main.go

package main

import (
	"context"
	file_transfer_api "file-transfer-api"
	"log/slog"
	"os"

	appgrpc "file-transfer-api/internal/handler/grpc"
	apphttp "file-transfer-api/internal/handler/http"
	"file-transfer-api/internal/infra"
	"file-transfer-api/internal/lifecycle"
	"file-transfer-api/internal/pkg/config"
	"file-transfer-api/internal/pkg/logger"
	"file-transfer-api/internal/usecase"
)

func main() {
	// 1. 設定とログの初期化
	cfg := config.Load()

	ctx := context.Background()

	// ---------------------------------------------------------
	// [1] システム基盤の準備
	// ---------------------------------------------------------

	// --- [1] 基盤準備（ログ・設定） ---
	// 🚀 独自の TraceHandler を噛ませる
	baseHandler := slog.NewJSONHandler(os.Stdout, nil)
	traceHandler := &logger.TraceHandler{Handler: baseHandler}
	// ログ出力を構造化（JSON）し、標準ロガーとして設定
	slog.SetDefault(slog.New(traceHandler))

	// =========================================================
	//  [2] インフラ層の一括初期化
	// =========================================================
	resources, err := infra.InitInfrastructure(ctx, file_transfer_api.MigrationFS, cfg)
	if err != nil {
		slog.Error("❌ Infrastructure initialization failed", "error", err)
		os.Exit(1)
	}
	// 💡 アプリ終了時（Runner.Run() を抜けた後）に、内包されたクリーンアップが一斉に逆順実行される
	defer resources.Cleanup()
	// ---------------------------------------------------------
	// [3] アプリケーション層（ドメインロジック）の構築
	// ---------------------------------------------------------

	// 3. ユースケースの初期化
	interactor := usecase.NewFileInteractor(resources.FileRepo, resources.MetadataRepo, resources.DataPipeline)

	// =========================================================
	// [4] 各プロトコルサーバーの構造体初期化 (依存注入)
	// =========================================================
	httpServer := apphttp.NewServer(interactor, resources.MetadataRepo, cfg)
	grpcServer := appgrpc.NewServer(interactor, cfg)

	// ---------------------------------------------------------
	// [5] SERVER_MODE に応じたインテリジェントな起動制御
	// ---------------------------------------------------------
	// ライフサイクル管理マネージャーに委ねる（ここで綺麗にブロッキング待機します）
	runner := lifecycle.NewApplicationRunner(httpServer, grpcServer, cfg)
	runner.Run()
}
