// internal/handler/grpc/server.go

package grpc

import (
	"file-transfer-api/internal/domain"
	"file-transfer-api/internal/handler/appmiddleware"
	filepb "file-transfer-api/internal/handler/grpc/pb"
	"file-transfer-api/internal/pkg/config"
	"log/slog"
	"net"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

type Server struct {
	server *grpc.Server
	port   string
}

func NewServer(interactor domain.FileUseCase, cfg *config.Config) *Server {
	grpcSrv := grpc.NewServer(
		grpc.StreamInterceptor(appmiddleware.TraceStreamServerInterceptor()),
	)

	// ヘルスチェック登録
	healthSrv := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcSrv, healthSrv)
	healthSrv.SetServingStatus("FileService", grpc_health_v1.HealthCheckResponse_SERVING)
	reflection.Register(grpcSrv)

	// 🚀 Reflectionを登録（grpcurl等からサービス一覧を見えるようにする）
	handler := NewGRPCFileHandler(interactor)
	filepb.RegisterFileServiceServer(grpcSrv, handler)

	return &Server{
		server: grpcSrv,
		port:   cfg.GRPCPort, // 💡 パース済みの安全な設定値を使用
	}
}

func (s *Server) Start() error {
	lis, err := net.Listen("tcp", ":"+s.port)
	if err != nil {
		return err
	}

	slog.Info("📡 Starting gRPC server", "port", s.port)
	return s.server.Serve(lis)
}

func (s *Server) GracefulStop() {
	slog.Info("🛑 Shutting down gRPC server gracefully...")
	s.server.GracefulStop()
}

func StartServer(interactor domain.FileUseCase) {
	grpcSrv := grpc.NewServer(
		grpc.StreamInterceptor(appmiddleware.TraceStreamServerInterceptor()),
	)

	healthSrv := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcSrv, healthSrv)
	healthSrv.SetServingStatus("FileService", grpc_health_v1.HealthCheckResponse_SERVING)

	reflection.Register(grpcSrv)

	handler := NewGRPCFileHandler(interactor)
	filepb.RegisterFileServiceServer(grpcSrv, handler)

	port := os.Getenv("GRPC_PORT")
	if port == "" {
		port = "50050" // デフォルト値
	}

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
