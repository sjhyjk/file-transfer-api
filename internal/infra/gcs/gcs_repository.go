package gcs

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"
)

// GCSRepository はGCS操作の実体を持つ構造体です
type GCSRepository struct {
	client     *storage.Client
	bucketName string
}

// NewGCSRepository は初期化関数です（main.goなどで呼び出します）
func NewGCSRepository(ctx context.Context, bucketName, keyFile string) (*GCSRepository, error) {
	var opts []option.ClientOption

	// keyFile が指定されている（ローカル環境など）場合のみファイルを使う
	if keyFile != "" {
		opts = append(opts, option.WithCredentialsFile(keyFile))
	}
	// keyFile が空の場合、storage.NewClient は自動的に環境（Cloud Runなど）の認証情報を使います

	// 2. opts... （三点リーダー）を使ってスライスを展開して渡す
	client, err := storage.NewClient(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("GCSクライアントの作成に失敗: %w", err)
	}
	return &GCSRepository{
		client:     client,
		bucketName: bucketName,
	}, nil
}

// Close は接続を閉じます
func (r *GCSRepository) Close() error {
	return r.client.Close()
}

// Save はデータをGCSに保存します（ここが並行処理の検証対象になります）
// Save は io.Reader からデータを読み取り、GCSへストリーム転送します
func (r *GCSRepository) Save(ctx context.Context, objectName string, data io.Reader) error {
	// 🚀 低レイヤーのログ：どのバケットに保存しようとしているか
	slog.DebugContext(ctx, "GCS upload starting", "bucket", r.bucketName, "object", objectName)

	wc := r.client.Bucket(r.bucketName).Object(objectName).NewWriter(ctx)

	// io.Copy を使うことで、メモリに溜め込まずにバケツリレーで転送できます
	// io.Copy を使うことで、ReaderからGCSのWriterへ効率よくデータを流し込めます
	if _, err := io.Copy(wc, data); err != nil {
		slog.ErrorContext(ctx, "GCS copy failed", "object", objectName, "error", err)
		return fmt.Errorf("GCSへのコピー失敗: %w", err)
	}

	if err := wc.Close(); err != nil {
		slog.ErrorContext(ctx, "GCS writer close failed", "object", objectName, "error", err)
		return fmt.Errorf("クローズ失敗: %w", err)
	}

	return nil
}

// Delete は指定されたオブジェクトをGCSから削除します（ロールバック用）
func (r *GCSRepository) Delete(ctx context.Context, objectName string) error {
	slog.WarnContext(ctx, "GCS object deleting (rollback)", "bucket", r.bucketName, "object", objectName)

	// GCSオブジェクトの削除実行
	if err := r.client.Bucket(r.bucketName).Object(objectName).Delete(ctx); err != nil {
		return fmt.Errorf("GCSからの削除失敗: %w", err)
	}

	return nil
}
