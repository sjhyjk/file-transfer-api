package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"file-transfer-api/internal/domain"
	"file-transfer-api/internal/infra"
	"file-transfer-api/internal/usecase"
)

func main() {
	ctx := context.Background()

	// 1. 環境変数から取得（設定されていなければデフォルト値を使用）
	bucketName := os.Getenv("BUCKET_NAME")
	if bucketName == "" {
		// 開発中の利便性のために、今のバケット名をデフォルトに設定
		bucketName = "file-transfer-bucket-syou-20240121"
	}

	keyFile := os.Getenv("GCP_KEY_FILE")
	if keyFile == "" {
		keyFile = "gcp-key.json"
	}

	// 2. Infrastructureの初期化
	repo, err := infra.NewGCSRepository(ctx, bucketName, keyFile)
	if err != nil {
		log.Fatalf("リポジトリの初期化に失敗: %v", err)
	}
	defer repo.Close()

	// 3. Usecaseの初期化（ここでInfraを注入する）
	interactor := usecase.NewFileInteractor(repo)

	// 4. テストデータの準備（3つのファイルを並行で送る準備）
	testFiles := []*domain.File{
		domain.NewFile("parallel-test-1.txt", 100, bytes.NewReader([]byte("Data 1"))),
		domain.NewFile("parallel-test-2.txt", 100, bytes.NewReader([]byte("Data 2"))),
		domain.NewFile("parallel-test-3.txt", 100, bytes.NewReader([]byte("Data 3"))),
	}

	// 5. 並行アップロードの実行と計測
	start := time.Now()
	fmt.Println("🚀 並行アップロードを開始します...")

	if err := interactor.UploadMultiple(ctx, testFiles); err != nil {
		log.Fatalf("アップロード中にエラーが発生: %v", err)
	}

	duration := time.Since(start)
	fmt.Printf("✅ すべてのアップロードが完了しました！ (計測時間: %v)\n", duration)
}
