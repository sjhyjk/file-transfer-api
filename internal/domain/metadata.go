package domain

import "time"

// FileMetadata はファイルに付随する属性定義です。
// 将来的なRAGパイプラインやドメイン固有の検索要件（日付、作成者、タグ等）に使用します。
type FileMetadata struct {
	FileName  string
	CreatedAt time.Time
	Tags      []string
	Source    string // e.g., "manual_upload", "system_generated"
}

// TransferStatus は転送状態を表すドメイン定数です。
// ドメインモデルの深化フェーズで、リトライポリシー等の判定に使用します。
type TransferStatus int

const (
	StatusPending TransferStatus = iota
	StatusProcessing
	StatusCompleted
	StatusFailed
)
