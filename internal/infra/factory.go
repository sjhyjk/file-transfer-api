package infra

import (
	"context"
	"fmt"
	"os"

	"file-transfer-api/internal/domain"
	"file-transfer-api/internal/infra/gcs"
)

// NewStorageRepository は環境変数に応じて適切なリポジトリを返します
func NewStorageRepository(ctx context.Context) (domain.FileRepository, error) {
	// 環境変数 STORAGE_TYPE で切り替え (デフォルトは GCS)
	storageType := os.Getenv("STORAGE_TYPE")

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
		if keyFile == "" {
			keyFile = "gcp-key.json"
		}

		return gcs.NewGCSRepository(ctx, bucketName, keyFile)
	}
}
