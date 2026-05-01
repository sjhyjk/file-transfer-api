package main

import (
	"bytes"
	"context"
	file_transfer_api "file-transfer-api" // ルートパッケージをインポート
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"file-transfer-api/internal/domain"
	"file-transfer-api/internal/handler"
	"file-transfer-api/internal/handler/appmiddleware"
	"file-transfer-api/internal/infra"
	"file-transfer-api/internal/infra/sql"
	"file-transfer-api/internal/pkg/logger"
	"file-transfer-api/internal/usecase"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	// 🚀 .env ファイルを読み込む（コメントや空行があっても賢く無視してくれます）
	if err := godotenv.Load(); err != nil {
		// .envがなくても環境変数があれば動くよう、ログ出力に留めるのが一般的
		slog.Info(".env file not found, using system environment variables")
	}

	ctx := context.Background()

	// ---------------------------------------------------------
	// [1] システム基盤の準備
	// ---------------------------------------------------------

	// --- [1] 基盤準備（ログ・設定） ---
	// 🚀 修正：独自の TraceHandler を噛ませる
	baseHandler := slog.NewJSONHandler(os.Stdout, nil)
	traceHandler := &logger.TraceHandler{Handler: baseHandler}
	// ログ出力を構造化（JSON）し、標準ロガーとして設定
	logger := slog.New(traceHandler)
	slog.SetDefault(logger)

	// 外部環境（Cloud Runやローカル環境変数）から設定を取得
	dbURL := os.Getenv("DATABASE_URL")
	dbType := os.Getenv("DB_TYPE")

	// 抽象化されたリポジトリを保持する変数
	// 1. 具体的な実装を受け取る変数を、domain層のインターフェース型として定義（DIPの徹底）
	var (
		fileRepo     domain.FileRepository
		metadataRepo domain.MetadataRepository
	)

	// ---------------------------------------------------------
	// [2] インフラ層（外部依存）の初期化
	// ---------------------------------------------------------

	// --- 2.1 メタデータ（データベース）層 ---
	// --- [DB/メタデータ層の初期化] ---
	if dbType == "INMEMORY" {
		slog.Info("💡 DB_TYPE is INMEMORY. Skipping PostgreSQL connections and migrations.")

		// Factory経由でインメモリリポジトリを取得
		var err error
		metadataRepo, err = infra.NewMetadataRepository(ctx)
		if err != nil {
			slog.Error("❌ Failed to init In-Memory repo", "error", err)
			os.Exit(1)
		}
	} else {
		// --- [ここ！] DB接続とマイグレーション ---
		slog.Info("🔌 Connecting to Cloud SQL and Running Migrations...")

		var sqlRepo *sql.Repository
		var lastErr error

		// 🚀 リトライロジックをここに配置
		maxRetries := 5
		for i := 1; i <= maxRetries; i++ {
			// A. DBマイグレーションの実行（接続テストも兼ねる）
			// sql.RunMigrations は migrations.go で定義する関数
			// 🚀 ルートで定義した MigrationFS をインフラ層に注入する
			lastErr = sql.RunMigrations(ctx, dbURL, file_transfer_api.MigrationFS)
			if lastErr == nil {
				// B. マイグレーション成功ならリポジトリ生成
				sqlRepo, lastErr = sql.NewRepository(ctx)
				if lastErr == nil {
					break // ✨ 両方成功したらループを抜ける
				}
			}

			// 失敗時のログ（まだリトライの可能性がある場合）
			slog.Warn("⚠️ DB not ready. Retrying...", "attempt", i, "max", maxRetries, "error", lastErr)
			time.Sleep(2 * time.Second) // 2秒待機してリトライ

			// ⚡ 最終リトライでも失敗した場合：ここがかつての「ガード節」の終着点
			if i == maxRetries {
				slog.Error("❌ DB接続に最終失敗しました", "error", lastErr)
				os.Exit(1)
			}
		}

		// ここに来るということは、必ずリトライのどこかで成功している
		defer sqlRepo.Close()
		slog.Info("🎉 Database is ready and migrated!")

		metadataRepo = sqlRepo
	}

	// --- 2.2 ストレージ（ファイル保存）層 ---
	// --- [ストレージ層の初期化] ---
	// Factoryを使用して環境に応じたリポジトリ（GCS or LOCAL）を生成
	repo, err := infra.NewStorageRepository(ctx)
	if err != nil {
		slog.Error("⚠️ ストレージリポジトリの初期化に失敗", "error", err)
		os.Exit(1) // 🚀 ここで止める！
	}
	fileRepo = repo
	// storage側もCloseが必要なインターフェースならここでdefer
	// defer fileRepo.Close()

	// ---------------------------------------------------------
	// [3] アプリケーション層（ドメインロジック）の構築
	// ---------------------------------------------------------

	// --- [3] ベンチマーク計測 (起動時に1回実行) ---
	// 3. ユースケースの初期化（具体的な実装をインターフェースに注入）
	// これにより、usecase側には「実体(infra)が何か」を隠したまま「機能(interface)」だけを渡せます
	interactor := usecase.NewFileInteractor(fileRepo, metadataRepo, nil)

	// ---------------------------------------------------------
	// [4] 動作検証（ベンチマーク計測）
	// ---------------------------------------------------------

	// 4. テストデータの準備（3つのファイルを並行で送る準備）
	// 並行処理の有効性を確認するためのテストデータ
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

	// =========================================================
	// 🚀 [5] Cloud Run / API サーバー用設定
	// =========================================================

	// --- [4] Echo サーバーの構築 ---
	e := echo.New()

	// 🚀 永田さんの Middleware を Echo 用にラップして登録
	// (後述の Adaptor を使うか、Echo 用に書き換えたものを使用)
	e.Use(echo.WrapMiddleware(appmiddleware.TraceMiddleware))
	e.Use(middleware.Recover()) // パニック時に落ちないように

	// ハンドラーの初期化
	fileHandler := handler.NewFileHandler(interactor)

	// 🚀 自動生成されたハンドラーを一括登録！
	// これにより YAML で定義した /files, /upload 等が紐付きます
	handler.RegisterHandlers(e, fileHandler)

	// 🚀 個別のヘルスチェック (OpenAPI外の自由なルート)
	e.GET("/health-check", func(c echo.Context) error {
		dbStatus := "OK"
		if metadataRepo == nil {
			dbStatus = "NG"
		}
		return c.String(http.StatusOK, fmt.Sprintf("✅ Running! DB: %s, Gain: %.2f%%", dbStatus, improvement))
	})

	// --- [5] 起動 ---
	// Cloud Run は環境変数 "PORT" を指定してくるため、それを取得
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // ローカル実行時のデフォルト
	}

	// すでに上で port := os.Getenv("PORT") (or "8080") と定義しているので、
	// それを使い回すのが安全です。
	slog.Info("📡 Starting server", "port", port)

	// 🚀 Echo スタイルの起動方法（こちらを推奨）
	if err := e.Start(":" + port); err != nil {
		slog.Error("サーバー起動失敗", "error", err)
		os.Exit(1)
	}
}
