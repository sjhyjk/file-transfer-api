// cmd/api/main.go

package main

import (
	"context"
	file_transfer_api "file-transfer-api"
	"log/slog"
	"os"

	"file-transfer-api/internal/domain"
	appgrpc "file-transfer-api/internal/handler/grpc"
	apprest "file-transfer-api/internal/handler/rest"
	"file-transfer-api/internal/infra"
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
	// 🚀 修正：独自の TraceHandler を噛ませる
	baseHandler := slog.NewJSONHandler(os.Stdout, nil)
	traceHandler := &logger.TraceHandler{Handler: baseHandler}
	// ログ出力を構造化（JSON）し、標準ロガーとして設定
	slog.SetDefault(slog.New(traceHandler))

	// 抽象化されたリポジトリを保持する変数
	// 1. 具体的な実装を受け取る変数を、domain層のインターフェース型として定義（DIPの徹底）
	var (
		fileRepo     domain.FileRepository
		metadataRepo domain.MetadataRepository
		dataPipeline domain.DataPipeline
		err          error
		dbCleanup    func() // 変数名を具体的にして重複回避
	)

	// ---------------------------------------------------------
	// [2] インフラ層（外部依存）の初期化
	// ---------------------------------------------------------

	// --- 2.1 メタデータ（データベース）層 ---

	// Factory経由でインメモリリポジトリを取得
	// 🚀 司令塔（Factory）に丸投げ。中身が DB かインメモリかは Factory が知っている。
	metadataRepo, dbCleanup, err = infra.NewMetadataRepository(ctx, file_transfer_api.MigrationFS)
	if err != nil {
		slog.Error("failed to init metadata repo", "error", err)
		os.Exit(1)
	}
	defer dbCleanup() // SQLなら切断、インメモリなら何もしない、が自動で切り替わる

	// --- 2.2 ストレージ（ファイル保存）層 ---
	// --- [ストレージ層の初期化] ---

	var storageCleanup func()

	// Factoryを使用して環境に応じたリポジトリ（GCS or LOCAL）を生成
	fileRepo, storageCleanup, err = infra.NewStorageRepository(ctx, cfg)
	if err != nil {
		slog.Error("failed to init storage repo", "error", err)
		os.Exit(1) // 🚀 ここで止める！
	}
	defer storageCleanup()

	// --- 2.3 パイプライン（Python通知）層 --- ★ 追加
	dataPipeline, err = infra.NewDataPipeline(ctx, cfg)
	if err != nil {
		slog.Error("failed to init pipeline", "error", err)
		// 通知が必須でないなら Exit しなくても良いが、今回は繋いでおく
	}

	// ---------------------------------------------------------
	// [3] アプリケーション層（ドメインロジック）の構築
	// ---------------------------------------------------------

	// 3. ユースケースの初期化
	interactor := usecase.NewFileInteractor(fileRepo, metadataRepo, dataPipeline)

	// =========================================================
	// 🚀 [4] 各プロトコルサーバーの起動
	// =========================================================

	// 1. gRPCサーバーの起動 (別ポート: 50051)
	go appgrpc.StartServer(interactor, "50051")

	// 2. HTTPサーバーの起動 (メインスレッド)
	// ポート取得や詳細な設定は rest パッケージが責任を持つ
	apprest.StartServer(interactor, metadataRepo)
}
