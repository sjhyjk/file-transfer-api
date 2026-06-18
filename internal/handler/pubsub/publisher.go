// internal/handler/pubsub/publisher.go

package pubsub

import (
	"context"
	"file-transfer-api/internal/pkg/logger"
	"fmt"
	"log/slog"

	"cloud.google.com/go/pubsub/v2"
)

type Publisher struct {
	client    *pubsub.Client
	projectID string
}

// NewPublisher は Publisher を初期化します
func NewPublisher(client *pubsub.Client, projectID string) *Publisher {
	return &Publisher{
		client:    client,
		projectID: projectID,
	}
}

// PublishWithTrace は Context から Trace ID を抽出し、Pub/Sub メッセージのメタデータに仕込んで発行します
func (p *Publisher) PublishWithTrace(ctx context.Context, topicID string, data []byte, attrs map[string]string) *pubsub.PublishResult {
	if attrs == nil {
		attrs = make(map[string]string)
	}

	// 🚀 Context から現在の Trace ID を抽出して Attributes に乗せる
	traceID := logger.FromContext(ctx)
	if traceID != "" {
		attrs["x-trace-id"] = traceID
	}

	// 💡 引数のアトリビュートから、ログ用の共通識別子を抽出
	tenantID := attrs["tenant_id"]
	eventType := attrs["event_type"]

	// 💡 トピックはプロジェクト名を含む Full FQN で指定
	topicFQN := fmt.Sprintf("projects/%s/topics/%s", p.projectID, topicID)

	// ✨ リクエスト開始の構造化ログ（HTTP/gRPC ハンドラーとキー名・レベルを完全一致）
	slog.InfoContext(ctx, "Processing upload request via Pub/Sub",
		"tenant_id", tenantID,
		"event_type", eventType,
		"topic", topicFQN,
	)

	publisher := p.client.Publisher(topicFQN)

	// 💡 メッセージの発行
	result := publisher.Publish(ctx, &pubsub.Message{
		Data:       data,
		Attributes: attrs, // ここで Trace ID が Python 側に引き継がれる
	})

	// 呼び出し元がこの result を使って非同期に待機（Get）できるように戻しつつ、
	// ここでは発行処理が正常にバックグラウンドのバッファに乗った証跡を記録します。
	slog.InfoContext(ctx, "✅ [PubSub v2] Message buffered successfully",
		"tenant_id", tenantID,
		"topic", topicID,
	)

	return result
}
