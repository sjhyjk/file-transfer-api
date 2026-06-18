// internal/infra/pipeline/pubsub_notifier.go

package pipeline

import (
	"context"
	"encoding/json"
	"file-transfer-api/internal/domain"
	"file-transfer-api/internal/pkg/logger"
	"fmt"
	"log/slog"
	"time"

	"cloud.google.com/go/pubsub/v2"
)

type PubSubPipeline struct {
	client    *pubsub.Client
	publisher *pubsub.Publisher
	projectID string
}

// NewPubSubPipeline は Pub/Sub 用のパイプラインを初期化します
func NewPubSubPipeline(ctx context.Context, projectID, topicID string) (*PubSubPipeline, error) {
	client, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("[PubSub] failed to create client (project_id=%s): %w", projectID, err)
	}

	publisher := client.Publisher(topicID)
	return &PubSubPipeline{client: client, publisher: publisher}, nil
}

// Close はリソースを安全に解放します (統一インターフェース)
func (p *PubSubPipeline) Close(ctx context.Context) error {
	// リソース解放用のクリーンアップ関数を定義
	slog.InfoContext(ctx, "🔌 [PubSub] Closing client connection...")
	// 💡 終了時は (t *Publisher) Stop() を呼び出してGoroutineをクリーンアップ
	p.publisher.Stop()
	if err := p.client.Close(); err != nil {
		return fmt.Errorf("[PubSub] failed to close connection (project_id=%s): %w", p.projectID, err)
	}
	return nil
}

// NotifyNewFile はドメインメタデータを JSON 化して Pub/Sub へ Publish します
func (p *PubSubPipeline) NotifyNewFile(ctx context.Context, meta *domain.FileMetadata) error {
	// 1. 🚀 Context から現在の Trace ID を抽出
	traceID := logger.FromContext(ctx)

	// 2. 開始ログ
	slog.InfoContext(ctx, "🚀 [PubSub] Sending notification to client",
		"topic", p.publisher.ID(),
		"file_id", meta.ID,
		"file_name", meta.FileName,
		"tenant_id", meta.TenantID,
		"tags", meta.Tags,
		"status", string(meta.Status),
		"source", meta.Source,
		"created_at", meta.CreatedAt.Format(time.RFC3339),
	)

	// 3. 送信用Context生成 (5秒タイムアウト)
	// 非同期発行の完了待ちにもコンテキストのタイムアウト制御を適用
	pubCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// 4. ペイロード構築
	payload := ToPythonRAGPayload(meta)

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("[PubSub] failed to marshal payload (file_id=%d, tenant_id=%s): %w", meta.ID, meta.TenantID, err)
	}

	// メッセージの構築
	msg := &pubsub.Message{
		Data: body,
		Attributes: map[string]string{
			"tenant_id":  meta.TenantID,
			"event_type": "file_uploaded",
			"x-trace-id": traceID,
		},
	}

	slog.InfoContext(ctx, "📢 [PubSub] Publishing message via client",
		"file_id", meta.ID,
	)

	// 5. 🚀 Publish（非同期でのバッファリング発行。裏で自動で効率よくバッチングされます）
	result := p.publisher.Publish(pubCtx, msg)

	// 同期的に書き込み完了を確実に待ち受ける（確実にトピックに届いたことを検証する）
	serverID, err := result.Get(pubCtx)
	if err != nil {
		return fmt.Errorf("[PubSub] failed to publish message (topic=%s, file_id=%d, tenant_id=%s): %w", p.publisher.ID(), meta.ID, meta.TenantID, err)
	}

	// 6. 完了ログ
	slog.InfoContext(ctx, "✅ [PubSub] Message published successfully",
		"file_id", meta.ID,
		"message_id", serverID,
	)
	return nil
}
