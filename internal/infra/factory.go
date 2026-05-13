// internal/infra/factory.go

package infra

import (
	"context"
	"embed"
	"fmt"
	"log/slog"
	"os"

	"file-transfer-api/internal/domain"
	"file-transfer-api/internal/infra/gcs"
	"file-transfer-api/internal/infra/local"
	"file-transfer-api/internal/infra/pipeline"
	"file-transfer-api/internal/infra/repository/inmemory"
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
func NewMetadataRepository(ctx context.Context, fs embed.FS) (domain.MetadataRepository, func(), error) {
	dbType := os.Getenv("DB_TYPE")
	dbURL := os.Getenv("DATABASE_URL")

	if dbType == "INMEMORY" {
		slog.InfoContext(ctx, "💡 DB_TYPE is INMEMORY. Skipping PostgreSQL connections and migrations.")
		// インメモリの場合は cleanup (Close) が不要なので、空の関数を返します
		var repo domain.MetadataRepository = inmemory.NewInMemoryRepository()
		return repo, func() {}, nil
	}

	// 🚀 実機DBの場合はリトライとマイグレーションを実行
	repo, err := sql.OpenWithRetry(ctx, dbURL, fs)
	if err != nil {
		// エラーログは OpenWithRetry 内で詳細に出ているのでラップして返す
		return nil, nil, fmt.Errorf("failed to initialize SQL repository: %w", err)
	}

	// 呼び出し側で close できるように cleanup 関数を返す
	return repo, func() { repo.Close() }, nil
}

func NewDataPipeline(ctx context.Context, cfg *config.Config) (domain.DataPipeline, error) {
	// 環境変数などで Python 側の URL を取得できるようにする
	pythonURL := os.Getenv("PYTHON_RAG_URL")
	if pythonURL == "" {
		// 開発用デフォルト
		pythonURL = "http://localhost:8000/ingest"
	}

	slog.InfoContext(ctx, "Initializing Python RAG Pipeline", "url", pythonURL)
	return pipeline.NewPythonPipeline(pythonURL), nil
}
