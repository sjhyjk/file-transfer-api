// internal/infra/pipeline/http_notifier.go

package pipeline

import (
	"bytes"
	"context"
	"encoding/json"
	"file-transfer-api/internal/domain"
	"file-transfer-api/internal/pkg/logger"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

type HttpPythonPipeline struct {
	endpoint   string
	httpClient *http.Client
}

func NewHttpPythonPipeline(endpoint string) *HttpPythonPipeline {
	return &HttpPythonPipeline{
		endpoint: endpoint,
		// 💡 DefaultClientを廃止し、明示的なタイムアウトを持つカスタムクライアントを生成
		httpClient: &http.Client{
			// 💡 個別のWithTimeoutで制御するため、クライアント全体のハードタイムアウトは少し長めに安全弁として設定
			Timeout: 10 * time.Second,
		},
	}
}

func (p *HttpPythonPipeline) NotifyNewFile(ctx context.Context, meta *domain.FileMetadata) error {
	// 1. Trace ID取得
	traceID := logger.FromContext(ctx)

	// 2. 開始ログ
	// trace_id は ctx から自動付与されるため引数には不要
	slog.InfoContext(ctx, "🚀 [HTTP] Sending notification to client",
		"endpoint", p.endpoint,
		"file_id", meta.ID,
		"file_name", meta.FileName,
		"tenant_id", meta.TenantID,
		"tags", meta.Tags,
		"status", string(meta.Status),
		"source", meta.Source,
		"created_at", meta.CreatedAt.Format(time.RFC3339),
	)

	// 3. 送信用Context生成 (5秒タイムアウト)
	httpCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// 4. ペイロード構築
	// 💡 map ではなく共通関数から型安全な構造体を取得
	payload := ToPythonRAGPayload(meta)

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("[HTTP] failed to marshal payload (file_id=%d, tenant_id=%s): %w", meta.ID, meta.TenantID, err)
	}

	req, err := http.NewRequestWithContext(httpCtx, "POST", p.endpoint, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("[HTTP] failed to create request (file_id=%d, tenant_id=%s): %w", meta.ID, meta.TenantID, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Trace-Id", traceID)

	// 📢 直前ログ
	slog.InfoContext(ctx, "📢 [HTTP] Sending request via client",
		"file_id", meta.ID,
	)

	// 5. 💡 作成したhttpClientを使用してリクエストを実行
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("[HTTP] remote request failed (endpoint=%s, file_id=%d, tenant_id=%s): %w", p.endpoint, meta.ID, meta.TenantID, err)
	}

	// 「データを終端まで読み捨てる ＆ 閉じる」をセットで行い、キープアライブを確実に有効化する
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	// 💡 エラー発生（200 OK以外）時、Python側が返したエラー内容をログに出す
	if resp.StatusCode != http.StatusOK {
		respBody, readErr := io.ReadAll(resp.Body)

		errorDetail := "failed to read response body"
		if readErr == nil && len(respBody) > 0 {
			errorDetail = string(respBody)
		}

		return fmt.Errorf(
			"[HTTP] failed to send request (endpoint=%s, status=%d, file_id=%d, tenant_id=%s): %s", p.endpoint, resp.StatusCode, meta.ID, meta.TenantID, errorDetail,
		)
	}

	// 6. 完了ログ
	slog.InfoContext(ctx, "✅ [HTTP] Notification sent successfully",
		"file_id", meta.ID,
		"trace_id", traceID,
	)
	return nil
}
