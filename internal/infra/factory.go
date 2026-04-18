package infra

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"file-transfer-api/internal/domain"
	"file-transfer-api/internal/infra/gcs"
)

// NewStorageRepository は環境変数に応じて適切なリポジトリを返します
func NewStorageRepository(ctx context.Context) (domain.FileRepository, error) {
	// 環境変数 STORAGE_TYPE で切り替え (デフォルトは GCS)
	storageType := os.Getenv("STORAGE_TYPE")

	if storageType == "" {
		storageType = "GCS" // デフォルト
	}

	// 🚀 導入ポイント：どのインフラを選択したか記録する
	slog.InfoContext(ctx, "Initializing storage repository", "type", storageType)

	switch storageType {
	case "S3":
		// 将来的にここに AWS S3 の初期化を書く
		return nil, fmt.Errorf("S3 repository is not implemented yet")

	default:
		// GCS の初期化
		bucketName := os.Getenv("BUCKET_NAME")
		if bucketName == "" {
			bucketName = "file-transfer-bucket-syou-20240121"
		}

		keyFile := os.Getenv("GCP_KEY_FILE")
		// ローカル環境（ファイルがある場合）のみ keyFile を設定し、
		// なければ空のまま NewGCSRepository に渡すようにします。
		// ※以前のステップで修正した「空でも動くNewGCSRepository」と組み合わせます。

		return gcs.NewGCSRepository(ctx, bucketName, keyFile)
	}
}
