// internal/infra/factory.go

package infra

import (
	"context"
	"embed"
	"fmt"
	"log/slog"

	"file-transfer-api/internal/domain"
	"file-transfer-api/internal/infra/gcs"
	"file-transfer-api/internal/infra/inmemory"
	"file-transfer-api/internal/infra/local"
	"file-transfer-api/internal/infra/pipeline"
	"file-transfer-api/internal/infra/sql"
	"file-transfer-api/internal/pkg/config"
)

// NewStorageRepository は環境変数に応じて適切なリポジトリを返します
func NewStorageRepository(ctx context.Context, cfg *config.Config) (domain.FileRepository, func(), error) {
	// config ですでに値がセットされているので、そのまま使うだけ (デフォルトは GCS)
	storageType := cfg.StorageType

	if storageType == "" {
		storageType = "GCS" // デフォルト
	}

	var repo domain.FileRepository
	var err error

	// 🚀 導入ポイント：どのインフラを選択したか記録する
	slog.InfoContext(ctx, "Initializing storage repository", "type", storageType)

	// 1. 生成ロジックのみを switch に書く
	switch storageType {
	case "LOCAL":
		path := cfg.LocalStoragePath

		slog.InfoContext(ctx, "Local storage path detected", "path", path)
		repo = local.NewLocalRepository(path)

	case "S3":
		// 将来的にここに AWS S3 の初期化を書く
		err = fmt.Errorf("S3 repository is not implemented yet")

	default:
		// cfg.BucketName にはデフォルト値か環境変数の値が必ず入っている
		repo, err = gcs.NewGCSRepository(ctx, cfg.BucketName)
	}

	// 2. 🚀 最後に共通でエラーチェックをする
	// これにより、どのストレージタイプを選んでも失敗時に必ずログが出ます
	if err != nil {
		slog.ErrorContext(ctx, "⚠️ ストレージリポジトリの初期化に失敗",
			"type", storageType,
			"error", err,
		)
		return nil, nil, err
	}

	slog.InfoContext(ctx, "✅ Storage repository initialized", "type", storageType)

	// 🚀 修正：現状は Close が不要でも、将来のために空の cleanup を返す
	cleanup := func() {
		slog.Info("Closing storage repository")
	}

	return repo, cleanup, nil
}

// NewMetadataRepository は環境に応じて DB 実装を切り替える
// 🚀 sql.OpenWithRetry をここで呼ぶことで main をクリーンにする
func NewMetadataRepository(ctx context.Context, fs embed.FS, cfg *config.Config) (domain.MetadataRepository, func(), error) {
	dbType := cfg.DBType
	dbURL := cfg.DBURL

	if dbType == "INMEMORY" {
		slog.InfoContext(ctx, "💡 DB_TYPE is INMEMORY. Skipping PostgreSQL connections and migrations.")
		// インメモリの場合は cleanup (Close) が不要なので、空の関数を返します
		repo := inmemory.NewInMemoryRepository()
		return repo, func() {}, nil
	}

	// 1. 🚀 分離したマイグレーションを先に実行
	if err := sql.RunMigrations(ctx, dbURL, fs); err != nil {
		return nil, nil, fmt.Errorf("failed to run database migrations: %w", err)
	}

	// 2. 🚀 分離したDB接続リトライを実行してプールを取得
	pool, err := sql.OpenWithRetry(ctx, dbURL)
	if err != nil {
		// エラーログは OpenWithRetry 内で詳細に出ているのでラップして返す
		return nil, nil, fmt.Errorf("failed to open database pool: %w", err)
	}

	// 3. プールをリポジトリに注入
	repo := sql.NewRepository(pool)

	// ✨ 成功ログはここに配置！「接続」と「マイグレーション」が両方完了したことを明示します
	slog.InfoContext(ctx, "🎉 Database is ready and migrated successfully!")

	cleanup := func() {
		slog.Info("🔌 Closing PostgreSQL connection pool...")
		repo.Close()
	}

	// 呼び出し側で close できるように cleanup 関数を返す
	return repo, cleanup, nil
}

func NewDataPipeline(ctx context.Context, cfg *config.Config) (domain.DataPipeline, error) {
	slog.InfoContext(ctx, "Initializing Data Pipeline", "type", cfg.PipelineType)

	switch cfg.PipelineType {
	case "GCP":
		// 🚀 次の「Pub/Subコミット」でここにGCP用の箱やエミュレータ実装を繋ぐ
		return nil, fmt.Errorf("GCP Pub/Sub pipeline is not implemented yet")

	case "REDIS":
		// 🚀 最後の「Redisコミット」でここにRedisキュー実装を繋ぐ
		return nil, fmt.Errorf("redis pipeline is not implemented yet")

	case "HTTP":
		fallthrough
	default:
		// 現在のHTTP通知（暫定）を返す
		slog.InfoContext(ctx, "Using HTTP Python RAG Pipeline", "url", cfg.PythonRAGURL)
		return pipeline.NewPythonPipeline(cfg.PythonRAGURL), nil
	}
}
