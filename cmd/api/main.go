package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"time"

	"file-transfer-api/internal/domain"
	"file-transfer-api/internal/infra"
	"file-transfer-api/internal/usecase"
)

func main() {
	ctx := context.Background()

	// 1. Factory を使ってリポジトリを生成（具象クラスを隠蔽）
	repo, err := infra.NewStorageRepository(ctx)
	if err != nil {
		log.Fatalf("リポジトリの初期化に失敗: %v", err)
	}

	// defer repo.Close() // 必要に応じてRepositoryインターフェースにCloseを定義

	// 3. Usecaseの初期化（ここでInfraを注入する）
	interactor := usecase.NewFileInteractor(repo) // repo が domain.FileRepository 型ならOK

	// 4. テストデータの準備（3つのファイルを並行で送る準備）
	testFiles := []*domain.File{
		domain.NewFile("parallel-test-1.txt", 100, bytes.NewReader([]byte("Data 1"))),
		domain.NewFile("parallel-test-2.txt", 100, bytes.NewReader([]byte("Data 2"))),
		domain.NewFile("parallel-test-3.txt", 100, bytes.NewReader([]byte("Data 3"))),
	}

	// 5. 並行アップロードの実行と計測
	// ---------------------------------------------------------
	// 検証1: シリアル（逐次）アップロードの実行と計測
	// ---------------------------------------------------------
	fmt.Println("\n--- [Phase 1] Serial Upload Start ---")
	startSerial := time.Now()

	// 新しく追加する Serial 用メソッドを呼び出す
	if err := interactor.UploadMultipleSerial(ctx, testFiles); err != nil {
		log.Fatalf("シリアルアップロード中にエラーが発生: %v", err)
	}

	durationSerial := time.Since(startSerial)
	fmt.Printf("✅ シリアル完了 (計測時間: %v)\n", durationSerial)

	// ---------------------------------------------------------
	// 検証2: 並行（Goroutine）アップロードの実行と計測
	// ---------------------------------------------------------
	fmt.Println("\n--- [Phase 2] Parallel Upload Start ---")
	startParallel := time.Now()

	// Parallel 用にリネームしたメソッドを呼び出す
	if err := interactor.UploadMultipleParallel(ctx, testFiles); err != nil {
		log.Fatalf("並行アップロード中にエラーが発生: %v", err)
	}

	durationParallel := time.Since(startParallel)
	fmt.Printf("✅ 並行完了 (計測時間: %v)\n", durationParallel)

	// ---------------------------------------------------------
	// 6. 検証結果の比較（設計上の判断材料として出力）
	// ---------------------------------------------------------
	fmt.Printf("\n📈 Performance Benchmark Results:\n")
	fmt.Printf("  Method A (Serial):   %v\n", durationSerial)
	fmt.Printf("  Method B (Parallel): %v\n", durationParallel)

	// パフォーマンス改善率の計算
	improvement := float64(durationSerial-durationParallel) / float64(durationSerial) * 100
	fmt.Printf("  Performance Gain:    %.2f%%\n", improvement)
}
