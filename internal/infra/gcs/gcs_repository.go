package gcs

import (
	"context"
	"fmt"
	"io"

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
	client, err := storage.NewClient(ctx, option.WithCredentialsFile(keyFile))
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
	wc := r.client.Bucket(r.bucketName).Object(objectName).NewWriter(ctx)

	// io.Copy を使うことで、メモリに溜め込まずにバケツリレーで転送できます
	// io.Copy を使うことで、ReaderからGCSのWriterへ効率よくデータを流し込めます
	if _, err := io.Copy(wc, data); err != nil {
		return fmt.Errorf("GCSへのコピー失敗: %w", err)
	}

	if err := wc.Close(); err != nil {
		return fmt.Errorf("クローズ失敗: %w", err)
	}

	return nil
}
