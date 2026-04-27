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
	Close() error // これで main.go の defer が動くようになる

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

// DataPipeline は、保存されたデータを後続の処理（RAGのインジェストなど）へ
// 渡すための抽象的な「出口」を定義します。
type DataPipeline interface {
	NotifyNewFile(ctx context.Context, fileName string) error
}

// MetadataRepository はDB永続化の抽象化
// 基盤エンジニアとして、特定のDB（Postgres等）に依存しないビジネスロジックを記述するために定義します。
type MetadataRepository interface {
	// 保存（新規作成）
	Create(ctx context.Context, record *FileMetadata) error
	// 状態更新（完了・失敗など）
	UpdateStatus(ctx context.Context, id int64, status TransferStatus) error
	// IDによる取得
	FindByID(ctx context.Context, id int64) (*FileMetadata, error)
	// SaveMetadata は新規レコードをDBに保存し、生成されたIDと作成日時を構造体に反映します。
	SaveMetadata(ctx context.Context, metadata *FileMetadata) error
	// FindAll はページネーション付きでメタデータ一覧を取得します
	// これにより、Usecase層は「何を検索するか」だけを伝え、
	// 「どうSQLを書くか」はInfra層が責任を持つという分離が明確になります。
	FindAll(ctx context.Context, query FileSearchQuery) ([]*FileMetadata, error)
}
