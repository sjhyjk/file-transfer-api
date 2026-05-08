// cmd/benchmark/main.go

package main

import (
	"bytes"
	"context"
	file_transfer_api "file-transfer-api"
	"fmt"
	"log"
	"log/slog"
	"os"
	"time"

	"file-transfer-api/internal/domain"
	"file-transfer-api/internal/infra"
	"file-transfer-api/internal/pkg/config"
	"file-transfer-api/internal/pkg/logger"
	"file-transfer-api/internal/usecase"
)

func main() {
	// 🚀 設定のロード（.env の読み込みも Load 内で行われます）
	cfg := config.Load()

	// 構造化ログの設定
	baseHandler := slog.NewJSONHandler(os.Stdout, nil)
	slog.SetDefault(slog.New(&logger.TraceHandler{Handler: baseHandler}))

	ctx := context.Background()

	var (
		fileRepo       domain.FileRepository
		metadataRepo   domain.MetadataRepository
		err            error
		dbCleanup      func()
		storageCleanup func()
	)

	// ---------------------------------------------------------
	// [1] インフラ層（検証用インメモリ・ローカル）の初期化
	// ---------------------------------------------------------
	// 1. MetadataRepo (ベンチマーク時は DB_TYPE=INMEMORY を環境変数に設定することを想定)

	// ※ 性能差を見るため、I/O負荷が少ないインメモリ構成を基本とします
	metadataRepo, dbCleanup, err = infra.NewMetadataRepository(ctx, file_transfer_api.MigrationFS)
	if err != nil {
		os.Exit(1)
	}
	defer dbCleanup()

	// 2. StorageRepo (cfg を渡す)
	fileRepo, storageCleanup, err = infra.NewStorageRepository(ctx, cfg)
	if err != nil {
		slog.Error("failed to init storage repo for benchmark", "error", err)
		os.Exit(1)
	}
	defer storageCleanup() // domain.FileRepository に Close がなくても、Factory が返してくれれば安心

	interactor := usecase.NewFileInteractor(fileRepo, metadataRepo, nil)

	// ---------------------------------------------------------
	// [2] テストデータの準備
	// ---------------------------------------------------------
	// 並行処理の有効性を確認するためのテストデータ
	f1, _ := domain.NewFile("parallel-test-1.txt", 100, bytes.NewReader([]byte("Data 1")))
	f2, _ := domain.NewFile("parallel-test-2.txt", 100, bytes.NewReader([]byte("Data 2")))
	f3, _ := domain.NewFile("parallel-test-3.txt", 100, bytes.NewReader([]byte("Data 3")))

	testFiles := []*domain.File{f1, f2, f3}

	// ---------------------------------------------------------
	// [3] シリアル（逐次）アップロードの実行と計測
	// ---------------------------------------------------------
	fmt.Println("\n--- [Phase 1] Serial Upload Start ---")
	startSerial := time.Now()

	if err := interactor.UploadMultipleSerial(ctx, testFiles); err != nil {
		log.Fatalf("シリアルアップロード中にエラーが発生: %v", err)
	}

	durationSerial := time.Since(startSerial)
	fmt.Printf("✅ シリアル完了 (計測時間: %v)\n", durationSerial)

	// ---------------------------------------------------------
	// [4] 並行（Goroutine）アップロードの実行と計測
	// ---------------------------------------------------------
	fmt.Println("\n--- [Phase 2] Parallel Upload Start ---")
	startParallel := time.Now()

	if err := interactor.UploadMultipleParallel(ctx, testFiles); err != nil {
		log.Fatalf("並行アップロード中にエラーが発生: %v", err)
	}

	durationParallel := time.Since(startParallel)
	fmt.Printf("✅ 並行完了 (計測時間: %v)\n", durationParallel)

	// ---------------------------------------------------------
	// [5] 検証結果の出力
	// ---------------------------------------------------------
	fmt.Printf("\n📈 Performance Benchmark Results:\n")
	fmt.Printf("  Method A (Serial):   %v\n", durationSerial)
	fmt.Printf("  Method B (Parallel): %v\n", durationParallel)

	improvement := float64(durationSerial-durationParallel) / float64(durationSerial) * 100
	fmt.Printf("  Performance Gain:    %.2f%%\n", improvement)
}
