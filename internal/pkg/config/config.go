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
	HTTPPort         string
	GRPCPort         string
	LocalStoragePath string
	ServerMode       string // "GRPC", "HTTP" の完全切り替え用
	PipelineType     string // "HTTP", "GRPC", "GCP", "REDIS" など
	PythonRAGURL     string // HTTP通知用URL
	PythonGRPCTarget string // Python側のgRPCサーバー宛先用 (例: "localhost:50051")
	GCPProjectID     string
	PubSubTopicID    string
}

// Load は設定を読み込みます
func Load() *Config {
	// .env を読み込む（失敗してもシステム環境変数があればOKなのでInfoに留める）
	if err := godotenv.Load(); err != nil {
		slog.Info(".env file not found or skipped", "error", err)
	}

	cfg := &Config{
		DBURL:            os.Getenv("DATABASE_URL"),
		DBType:           os.Getenv("DB_TYPE"),
		StorageType:      os.Getenv("STORAGE_TYPE"),
		BucketName:       getEnv("BUCKET_NAME", "file-transfer-bucket-syou-20240121"),
		HTTPPort:         getEnv("HTTP_PORT", "8080"), // 💡 環境変数に合わせて取得
		GRPCPort:         getEnv("GRPC_PORT", "50050"),
		LocalStoragePath: getEnv("LOCAL_STORAGE_PATH", "/app/storage"),
		ServerMode:       getEnv("SERVER_MODE", "BOTH"),   // 💡 デフォルトは片方に絞るか、明確に"BOTH"にする
		PipelineType:     getEnv("PIPELINE_TYPE", "HTTP"), // デフォルトは現在のHTTP
		PythonRAGURL:     getEnv("PYTHON_RAG_URL", "http://localhost:8081/ingest"),
		PythonGRPCTarget: getEnv("PYTHON_GRPC_TARGET", "rag-worker-grpc:50051"), // 🚀 デフォルトの宛先
		GCPProjectID:     getEnv("GCP_PROJECT_ID", "local-project"),
		PubSubTopicID:    getEnv("PUBSUB_TOPIC_ID", "file-ingest-topic"),
	}

	// 🚀 堅牢性向上のためのガードロジック (Fail-Fast)
	if cfg.ServerMode == "BOTH" && cfg.HTTPPort == cfg.GRPCPort {
		slog.Error("❌ 設定エラー: SERVER_MODE=BOTH の時、HTTP_PORT と GRPC_PORT に同じポートは指定できません。",
			"http_port", cfg.HTTPPort,
			"grpc_port", cfg.GRPCPort,
		)
		os.Exit(1)
	}

	return cfg
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
