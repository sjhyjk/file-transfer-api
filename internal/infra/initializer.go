// internal/infra/initializer.go

package infra

import (
	"context"
	"embed"
	"file-transfer-api/internal/domain"
	"file-transfer-api/internal/pkg/config"
	"fmt"
	"log/slog"
)

type InfraResources struct {
	FileRepo     domain.FileRepository
	MetadataRepo domain.MetadataRepository
	DataPipeline domain.DataPipeline
	Cleanup      func() // 💡 すべてのクリーンアップを一括で行う関数
}

// InitInfrastructure は外部依存（DB, ストレージ, パイプライン）を順番に初期化します
func InitInfrastructure(ctx context.Context, fs embed.FS, cfg *config.Config) (*InfraResources, error) {
	// 1. メタデータリポジトリの初期化
	metadataRepo, dbCleanup, err := NewMetadataRepository(ctx, fs, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to init metadata repo: %w", err)
	}

	// 2. ストレージリポジトリの初期化
	fileRepo, storageCleanup, err := NewStorageRepository(ctx, cfg)
	if err != nil {
		dbCleanup() // 💡 手前で開いたリソースを安全に閉じる
		return nil, fmt.Errorf("failed to init storage repo: %w", err)
	}

	// 3. パイプラインの初期化
	dataPipeline, pipelineCleanup, err := NewDataPipeline(ctx, cfg)
	if err != nil {
		slog.Error("failed to init pipeline", "error", err)
		// 必須でなければ続行（通知失敗でもアプリ自体は動かすため）
	}

	// すべてのクリーンアップをカプセル化した関数を定義
	combinedCleanup := func() {
		if dbCleanup != nil {
			dbCleanup()
		}
		if storageCleanup != nil {
			storageCleanup()
		}
		if pipelineCleanup != nil {
			pipelineCleanup()
		}
	}

	// 具体的な実装を受け取る変数を、domain層のインターフェース型として定義（DIPの徹底）
	return &InfraResources{
		MetadataRepo: metadataRepo,
		FileRepo:     fileRepo,
		DataPipeline: dataPipeline,
		Cleanup:      combinedCleanup,
	}, nil
}
