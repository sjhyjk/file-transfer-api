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
	payload := map[string]interface{}{
		"file_id":   meta.ID,
		"file_name": meta.FileName,
		"tenant_id": meta.TenantID,
		"tags":      meta.Tags,
		"status":    meta.Status,
		"source":    meta.Source,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.endpoint, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("python rag returned status: %d", resp.StatusCode)
	}
	return nil
}
