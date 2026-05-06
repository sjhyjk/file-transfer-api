package grpc

import (
	"file-transfer-api/internal/domain"
	filepb "file-transfer-api/internal/handler/grpc/pb"
	"file-transfer-api/internal/usecase"
	"io"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GRPCFileHandler struct {
	// protocが生成する未実装エラー防止用の構造体を埋め込み
	filepb.UnimplementedFileServiceServer
	interactor *usecase.FileInteractor
}

func NewGRPCFileHandler(interactor *usecase.FileInteractor) *GRPCFileHandler {
	return &GRPCFileHandler{interactor: interactor}
}

func (h *GRPCFileHandler) UploadFile(stream filepb.FileService_UploadFileServer) error {
	// 1. パイプを作成
	pr, pw := io.Pipe()

	// 2. 最初のメッセージ（Metadata）を待機
	req, err := stream.Recv()
	if err != nil {
		return err
	}
	meta := req.GetMetadata()

	// 3. 非同期でストリームを読み込み、パイプに書き込む
	go func() {
		defer pw.Close()
		for {
			req, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				pw.CloseWithError(err)
				return
			}
			if chunk := req.GetChunk(); chunk != nil {
				pw.Write(chunk)
			}
		}
	}()

	// 4. 既存の Usecase を呼び出す
	// ストリームから流れてくる pr (io.Reader) をそのまま domain.File に渡す
	f := domain.NewFile(meta.Filename, 0, pr) // サイズ不明の場合は 0 またはメタデータから取得

	// 既存のロジックを再利用（DIPの恩恵）
	err = h.interactor.UploadMultipleParallel(stream.Context(), []*domain.File{f})
	if err != nil {
		// HTTPの500に相当する gRPC Internal エラーを返す
		return status.Errorf(codes.Internal, "upload failed: %v", err)
	}

	// ✨ 最終的な戻り値：成功時はクライアントにステータスを返して閉じる
	return stream.SendAndClose(&filepb.UploadFileResponse{
		Status:  "success",
		Message: "File uploaded successfully via gRPC stream",
	})
}
