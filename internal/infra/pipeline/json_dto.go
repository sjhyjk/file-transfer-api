// internal/infra/pipeline/json_dto.go

package pipeline

import (
	"file-transfer-api/internal/domain"
	"time"
)

// PythonRAGIngestPayload は HTTP と Pub/Sub で共通して送信する JSON のデータ構造を定義します
type PythonRAGIngestPayload struct {
	FileID    int64    `json:"file_id"`
	FileName  string   `json:"file_name"`
	TenantID  string   `json:"tenant_id"`
	Tags      []string `json:"tags"`
	Status    string   `json:"status"`
	Source    string   `json:"source"`
	CreatedAt string   `json:"created_at"`
}

// ToPythonRAGPayload はドメインモデルを HTTP/PubSub 共通の JSON ペイロード構造体に変換します
func ToPythonRAGPayload(meta *domain.FileMetadata) *PythonRAGIngestPayload {
	if meta == nil {
		return nil
	}
	return &PythonRAGIngestPayload{
		FileID:    meta.ID,
		FileName:  meta.FileName,
		TenantID:  meta.TenantID,
		Tags:      meta.Tags,
		Status:    string(meta.Status),
		Source:    meta.Source,
		CreatedAt: meta.CreatedAt.Format(time.RFC3339), // 💡 ここでフォーマットを強制統一
	}
}
