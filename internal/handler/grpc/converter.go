// internal/handler/grpc/converter.go

package grpc

import (
	"file-transfer-api/internal/domain"
	pb "file-transfer-api/internal/handler/grpc/pb"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// ToProtoStatus はドメイン層のステータス文字列を proto の Enum 型に変換します（送信・返却用）
func ToProtoStatus(domainStatus domain.TransferStatus) pb.TransferStatus {
	switch domainStatus {
	case domain.StatusPending:
		return pb.TransferStatus_TRANSFER_STATUS_PENDING
	case domain.StatusProcessing:
		return pb.TransferStatus_TRANSFER_STATUS_PROCESSING
	case domain.StatusCompleted:
		return pb.TransferStatus_TRANSFER_STATUS_COMPLETED
	case domain.StatusFailed:
		return pb.TransferStatus_TRANSFER_STATUS_FAILED
	default:
		return pb.TransferStatus_TRANSFER_STATUS_UNSPECIFIED
	}
}

// ToDomainStatus は proto の Enum 型をドメイン層のステータス文字列に変換します（リクエスト受信時用）
func ToDomainStatus(protoStatus pb.TransferStatus) domain.TransferStatus {
	switch protoStatus {
	case pb.TransferStatus_TRANSFER_STATUS_PENDING:
		return domain.StatusPending
	case pb.TransferStatus_TRANSFER_STATUS_PROCESSING:
		return domain.StatusProcessing
	case pb.TransferStatus_TRANSFER_STATUS_COMPLETED:
		return domain.StatusCompleted
	case pb.TransferStatus_TRANSFER_STATUS_FAILED:
		return domain.StatusFailed
	default:
		return ""
	}
}

// ToProtoFileIngestEvent はドメインの FileMetadata を gRPC 通信用の FileIngestEvent にマッピングします
func ToProtoFileIngestEvent(meta *domain.FileMetadata) *pb.FileIngestEvent {
	if meta == nil {
		return nil
	}
	return &pb.FileIngestEvent{
		FileId:    meta.ID,
		FileName:  meta.FileName,
		TenantId:  meta.TenantID,
		Tags:      meta.Tags,
		Source:    meta.Source,
		Status:    ToProtoStatus(meta.Status),
		CreatedAt: timestamppb.New(meta.CreatedAt),
	}
}
