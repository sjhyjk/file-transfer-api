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
	"file-transfer-api/internal/infra"
	"file-transfer-api/internal/infra/repository/sql"
	"file-transfer-api/internal/usecase"
)

func main() {
	ctx := context.Background()

	// ❌ 削除: os.Setenv("DATABASE_URL", ...) は行わず、環境変数から取得する
	// 0. DATABASE_URL をセット（IPとパスワードはご自身のものに）
	//os.Setenv("DATABASE_URL", "postgres://app_user:cits9999@8.229.68.93:5432/transfer_metadata?sslmode=disable")

	// --- [追加] JSONロガーの初期化 ---
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// 0. DATABASE_URL を取得
	dbURL := os.Getenv("DATABASE_URL")

	// --- [ここ！] DB接続とマイグレーション ---
	slog.Info("🔌 Connecting to Cloud SQL and Running Migrations...")

	// A. まずはマイグレーションを実行
	// sql.RunMigrations は migrations.go で定義する関数
	// 🚀 ルートで定義した MigrationFS をインフラ層に注入する
	if err := sql.RunMigrations(ctx, dbURL, file_transfer_api.MigrationFS); err != nil {
		slog.Error("❌ Migration failed（起動を中止します）", "error", err)
		os.Exit(1)
	}

	// B. リポジトリを初期化（中でPing確認をしているはず）
	sqlRepo, err := sql.NewRepository(ctx)

	// 異常系を先に処理して終わらせる（ガード節）
	if err != nil {
		slog.Error("❌ DB接続失敗（起動を中止します）", "error", err)
		os.Exit(1) // ここで確実に止まる
	}

	// ここに来るということは、必ず成功している（elseがいらない）
	defer sqlRepo.Close()
	slog.Info("🎉 Cloud SQL への接続に成功しました！")

	// 1. Factory を使ってリポジトリを生成（具象クラスを隠蔽）
	repo, err := infra.NewStorageRepository(ctx)
	var initError error
	if err != nil {
		slog.Error("⚠️ ストレージリポジトリの初期化に失敗", "error", err)
		initError = err // エラーを保持しておく
	}

	// defer repo.Close() // 必要に応じてRepositoryインターフェースにCloseを定義

	// 1. 具体的な実装を受け取る変数を、domain層のインターフェース型として定義（DIPの徹底）
	var (
		fileRepo     domain.FileRepository
		metadataRepo domain.MetadataRepository
	)

	// 2. 作成したインスタンスをインターフェース型に代入（これで抽象化される）
	fileRepo = repo
	metadataRepo = sqlRepo

	// 3. Usecaseの初期化時には、インターフェース型の変数として渡す
	// これにより、usecase側には「実体(infra)が何か」を隠したまま「機能(interface)」だけを渡せます
	interactor := usecase.NewFileInteractor(fileRepo, metadataRepo, nil)

	// ★ 追加：Handler の初期化
	fileHandler := handler.NewFileHandler(interactor) // 👈 追加

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

	// =========================================================
	// 🚀 [New] Cloud Run / API サーバー用設定
	// =========================================================

	// Cloud Run は環境変数 "PORT" を指定してくるため、それを取得
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // ローカル実行時のデフォルト
	}

	// ヘルスチェック用のエンドポイント（Cloud Run の起動確認に必要）
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if initError != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "❌ Initialization Error: %v\n", initError)
			fmt.Fprintf(w, "Check Cloud Run Env Vars (STORAGE_TYPE etc.)")
			return
		}
		// DB接続状況もヘルスチェックに含めると「基盤」っぽくなります
		dbStatus := "OK"
		if sqlRepo == nil {
			dbStatus = "NG"
		}
		fmt.Fprintf(w, "✅ Running! DB Status: %s, Gain: %.2f%%", dbStatus, improvement)
	})

	// ★ 追加：メタデータ一覧取得エンドポイント
	// これにより GET /files?limit=20&offset=0 が有効になります
	http.HandleFunc("/files", fileHandler.HandleListFiles) // 👈 追加

	// アップロード用のエンドポイント（将来的にここへ POST する）
	http.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request) {
		// ここに interactor.UploadMultipleParallel を呼ぶロジックを移譲予定
		fmt.Fprintln(w, "Upload endpoint reached")
	})

	// すでに上で port := os.Getenv("PORT") (or "8080") と定義しているので、
	// それを使い回すのが安全です。
	slog.Info("📡 Starting server", "port", port)

	// サーバーを起動（ここでプログラムが終了せずに待機状態になります）
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		slog.Error("サーバー起動失敗", "error", err)
		os.Exit(1)
	}
}
