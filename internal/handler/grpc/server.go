// internal/handler/grpc/server.go

package grpc

import (
	"file-transfer-api/internal/domain"
	filepb "file-transfer-api/internal/handler/grpc/pb"
	"log/slog"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

func StartServer(interactor domain.FileUseCase, port string) {
	grpcSrv := grpc.NewServer()

	// ヘルスチェック登録
	healthSrv := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcSrv, healthSrv)
	healthSrv.SetServingStatus("FileService", grpc_health_v1.HealthCheckResponse_SERVING)

	// 🚀 Reflectionを登録（grpcurl等からサービス一覧を見えるようにする）
	reflection.Register(grpcSrv)

	handler := NewGRPCFileHandler(interactor)
	filepb.RegisterFileServiceServer(grpcSrv, handler)

	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		slog.Error("gRPC listener failed", "error", err)
		return
	}

	slog.Info("📡 Starting gRPC server", "port", port)
	if err := grpcSrv.Serve(lis); err != nil {
		slog.Error("gRPC server failed", "error", err)
	}
}
