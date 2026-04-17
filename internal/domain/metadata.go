package domain

import "time"

// FileMetadata はファイルに付随する属性定義（Entity）です。
// 将来的なRAGパイプラインやドメイン固有の検索要件（日付、作成者、タグ等）に使用します。
type FileMetadata struct {
	ID        int64
	FileName  string
	FileSize  int64
	Status    TransferStatus
	Source    string   // e.g., "manual_upload", "system_generated"
	Tags      []string // ★ RAG等の検索要件のため
	CreatedAt time.Time
	UpdatedAt time.Time
}

// TransferStatus は転送状態を表すドメイン定数です。
// ドメインモデルの深化フェーズで、リトライポリシー等の判定に使用します。
type TransferStatus string

const (
	StatusPending    TransferStatus = "pending"
	StatusProcessing TransferStatus = "processing"
	StatusCompleted  TransferStatus = "completed"
	StatusFailed     TransferStatus = "failed"
)
