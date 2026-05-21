// internal/infra/pipeline/grpc_notifier.go

package pipeline

import (
	"context"
	"file-transfer-api/internal/domain"
	"log/slog"
)

type GrpcPythonPipeline struct {
	target string
}

func NewGrpcPythonPipeline(target string) *GrpcPythonPipeline {
	return &GrpcPythonPipeline{target: target}
}

// NotifyNewFile は、将来的にPythonのgRPCサーバーへ型安全に通知を送信するように作り込みます
func (g *GrpcPythonPipeline) NotifyNewFile(ctx context.Context, meta *domain.FileMetadata) error {
	slog.InfoContext(ctx, "🚀 [Mock] Sending gRPC notification to Python RAG",
		"target", g.target,
		"file_id", meta.ID,
		"tenant_id", meta.TenantID,
	)
	return nil
}
