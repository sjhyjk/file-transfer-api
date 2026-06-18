// internal/infra/pipeline/grpc_notifier.go

package pipeline

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"file-transfer-api/internal/domain"
	appgrpc "file-transfer-api/internal/handler/grpc"
	pb "file-transfer-api/internal/handler/grpc/pb"
	"file-transfer-api/internal/pkg/logger"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

type GrpcPythonPipeline struct {
	target string
	conn   *grpc.ClientConn
}

func NewGrpcPythonPipeline(target string) (*GrpcPythonPipeline, error) {
	// 最新の推奨 API `grpc.NewClient` を使用してコネクションを生成 (DialContext & WithBlock は非推奨)
	// ※ ローカル検証用のため、安全な通信（TLS）ではなく Insecure を指定しています
	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("[gRPC] failed to create client (target=%s): %w", target, err)
	}
	return &GrpcPythonPipeline{target: target, conn: conn}, nil
}

// Close はアプリ終了時に呼ばれる (メソッド内での毎回Closeを廃止)
func (g *GrpcPythonPipeline) Close(ctx context.Context) error {
	slog.InfoContext(ctx, "🔌 [gRPC] Closing client connection...")
	if err := g.conn.Close(); err != nil {
		return fmt.Errorf("[gRPC] failed to close connection (target=%s): %w", g.target, err)
	}
	return nil
}

// NotifyNewFile は、PythonのgRPCサーバーへ型安全に通知を送信します
func (g *GrpcPythonPipeline) NotifyNewFile(ctx context.Context, meta *domain.FileMetadata) error {
	// 1. Trace ID取得
	traceID := logger.FromContext(ctx)

	// 2. 開始ログ
	slog.InfoContext(ctx, "🚀 [gRPC] Sending notification to client",
		"target", g.target,
		"file_id", meta.ID,
		"file_name", meta.FileName,
		"tenant_id", meta.TenantID,
		"tags", meta.Tags,
		"status", string(meta.Status),
		"source", meta.Source,
		"created_at", meta.CreatedAt.Format(time.RFC3339),
	)

	// 3. 送信用Context生成 (5秒タイムアウト)
	rpcCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// 💡 gRPC特有のメタデータ注入
	rpcCtx = metadata.NewOutgoingContext(rpcCtx, metadata.Pairs("x-trace-id", traceID))

	// 2. 自動生成されたコードからクライアントを生成
	client := pb.NewFileServiceClient(g.conn) // protoで定義したサービス名に依存

	// 3. protoの定義（型定義）に基づいたリクエストメッセージの組み立て
	// 要求メッセージの組み立て（protoの定義に合わせてフィールドをマッピング）
	// 💡 作成したコンバーターを呼び出してリクエストを生成。Notifier内部が劇的にシンプルに！
	req := appgrpc.ToProtoFileIngestEvent(meta)

	// 🚀 リクエスト実行直前の進捗ログ
	slog.InfoContext(ctx, "📢 [gRPC] Sending request via client",
		"file_id", meta.ID,
	)

	// 5. Python側のgRPCエンドポイント（grpc_handler.py 等）をリモート呼び出し
	res, err := client.NotifyIngest(rpcCtx, req)
	if err != nil {
		return fmt.Errorf("[gRPC] failed to send rpc (target=%s, method=NotifyIngest, file_id=%d, tenant_id=%s): %w", g.target, meta.ID, meta.TenantID, err)
	}

	slog.InfoContext(ctx, "✅ [gRPC] Notification sent successfully",
		"file_id", meta.ID,
		"remote_status", res.GetStatus(),
	)
	return nil
}
