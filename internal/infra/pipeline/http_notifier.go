// internal/infra/pipeline/http_notifier.go

package pipeline

import (
	"bytes"
	"context"
	"encoding/json"
	"file-transfer-api/internal/domain"
	"fmt"
	"net/http"
)

type PythonPipeline struct {
	endpoint string
}

func NewPythonPipeline(endpoint string) *PythonPipeline {
	return &PythonPipeline{endpoint: endpoint}
}

func (p *PythonPipeline) NotifyNewFile(ctx context.Context, meta *domain.FileMetadata) error {
	// 検討していたペイロード
	payload := map[string]interface{}{
		"file_id":   meta.ID,
		"file_name": meta.FileName,
		"tags":      meta.Tags,
		"status":    meta.Status,
		"source":    meta.Source,
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, "POST", p.endpoint, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("python rag returned status: %d", resp.StatusCode)
	}
	return nil
}
