// internal/pkg/config/config.go

package config

import (
	"log/slog"
	"os"

	"github.com/joho/godotenv"
)

// Config はアプリケーション全体の設定を保持する構造体
// 将来的に「数値のバリデーション」などもここで行えます
type Config struct {
	DBURL            string
	DBType           string
	StorageType      string
	BucketName       string
	Port             string
	LocalStoragePath string
	ServerMode       string // "GRPC", "REST" の完全切り替え用
	PipelineType     string // "HTTP", "GRPC", "GCP", "REDIS" など
	PythonRAGURL     string // HTTP通知用URL
	PythonGRPCTarget string // Python側のgRPCサーバー宛先用 (例: "localhost:50051")
}

// Load は設定を読み込みます
func Load() *Config {
	// .env を読み込む（失敗してもシステム環境変数があればOKなのでInfoに留める）
	if err := godotenv.Load(); err != nil {
		slog.Info(".env file not found or skipped", "error", err)
	}

	return &Config{
		DBURL:            os.Getenv("DATABASE_URL"),
		DBType:           os.Getenv("DB_TYPE"),
		StorageType:      os.Getenv("STORAGE_TYPE"),
		BucketName:       getEnv("BUCKET_NAME", "file-transfer-bucket-syou-20240121"),
		Port:             getEnv("PORT", "8080"),
		LocalStoragePath: getEnv("LOCAL_STORAGE_PATH", "/app/storage"),
		ServerMode:       getEnv("SERVER_MODE", "BOTH"),   // 🚀 デフォルトは安全のため両方起動
		PipelineType:     getEnv("PIPELINE_TYPE", "HTTP"), // デフォルトは現在のHTTP
		PythonRAGURL:     getEnv("PYTHON_RAG_URL", "http://localhost:8000/ingest"),
		PythonGRPCTarget: getEnv("PYTHON_GRPC_TARGET", "rag-worker-grpc:50051"), // 🚀 デフォルトの宛先
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
