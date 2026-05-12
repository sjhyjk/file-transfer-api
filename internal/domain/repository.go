// internal/domain/repository.go

package domain

import (
	"context"
	"io"
)

// FileRepository は外部ストレージ（GCS/S3等）操作の抽象化
// domain層に置くことで、全ての層から参照可能になります
type FileRepository interface {
	Save(ctx context.Context, name string, data io.Reader) error
	Delete(ctx context.Context, name string) error
	// Close() error // これで main.go の defer が動くようになる
	// Close() は Factory 側の cleanup 関数で担保するため、
	// 純粋にストレージ操作のメソッドだけに絞るのも手です。

	// 今後の深化：ビジネスルールに基づくバッチ処理やリトライの抽象化
	// FindAllByStatus(ctx context.Context, status TransferStatus) ([]*File, error)
}

// FileSearchQuery はフィルタリング条件をカプセル化した構造体です。
// 将来的に「日付範囲」や「ファイルサイズ」が増えても、インターフェースのシグネチャを壊さずに済みます。
type FileSearchQuery struct {
	Tags   []string // タグによる絞り込み（複数指定はAND想定）
	Limit  int
	Offset int
}

// MetadataRepository はDB永続化の抽象化
// 基盤エンジニアとして、特定のDB（Postgres等）に依存しないビジネスロジックを記述するために定義します。
type MetadataRepository interface {
	// 1. 最初の作成（IDを発行し、ステータスを Pending にする）
	Create(ctx context.Context, record *FileMetadata) error

	// 2. 途中経過や最終結果の保存（冪等性を持たせた更新）
	// SaveMetadata は新規レコードをDBに保存し、生成されたIDと作成日時を構造体に反映します。
	SaveMetadata(ctx context.Context, metadata *FileMetadata) error

	// 状態更新（完了・失敗など）
	UpdateStatus(ctx context.Context, id int64, status TransferStatus) error
	// IDによる取得
	FindByID(ctx context.Context, id int64) (*FileMetadata, error)
	// FindAll はページネーション付きでメタデータ一覧を取得します
	FindAll(ctx context.Context, query FileSearchQuery) ([]*FileMetadata, error)
}

// DataPipeline (RAG等の外部通知用)
type DataPipeline interface {
	NotifyNewFile(ctx context.Context, meta *FileMetadata) error
}

// FileUseCase (アプリケーション層への入り口)
type FileUseCase interface {
	UploadSingle(ctx context.Context, name string, size int64, content io.Reader) error
	UploadMultipleParallel(ctx context.Context, files []*File) error
	FetchMetadataList(ctx context.Context, tags []string, limit, offset int) ([]*FileMetadata, error)
}
