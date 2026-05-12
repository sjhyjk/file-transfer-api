// internal/handler/grpc/file_handler_grpc.go

package grpc

import (
	"file-transfer-api/internal/domain"
	filepb "file-transfer-api/internal/handler/grpc/pb"
	"io"
	"log/slog"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GRPCFileHandler struct {
	// protocが生成する未実装エラー防止用の構造体を埋め込み
	filepb.UnimplementedFileServiceServer
	interactor domain.FileUseCase
}

func NewGRPCFileHandler(interactor domain.FileUseCase) *GRPCFileHandler {
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
	// 🚀 クライアントから送られた期待サイズを検証
	// domain.NewFile 内で「1GB以上はエラー」等のロジックを動かす
	if _, err := domain.NewFile(meta.Filename, meta.ExpectedSize, pr); err != nil {
		return status.Errorf(codes.InvalidArgument, "Validation failed: %v", err)
	}

	// ✨ リクエスト開始のログ（コンテキストの付与）
	slog.Info("gRPC upload stream started",
		"filename", meta.Filename,
		"trace_id", stream.Context().Value("trace_id"), // slog 拡張に合わせたキー
	)

	// 3. 非同期でストリームを読み込み、パイプに書き込む
	go func() {
		// pw.Close() の戻り値を無視せず、必要なら CloseWithError に集約
		defer func() {
			if closeErr := pw.Close(); closeErr != nil {
				// slog 拡張を利用してエラーを記録
				// 必要に応じて logger パッケージをインポートしてください
				slog.Error("failed to close pipe writer in gRPC stream", "error", closeErr)
			}
		}()

		for {
			req, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				// pr 側にエラーを伝播させる（errcheck対象外）
				_ = pw.CloseWithError(err)
				return
			}

			if chunk := req.GetChunk(); chunk != nil {
				// 【修正】pw.Write のエラーチェックを追加
				if _, writeErr := pw.Write(chunk); writeErr != nil {
					slog.Warn("pipe write interrupted", "error", writeErr) // 異常終了の予兆として記録
					// 書き込みエラー（pr側が閉じられた等）が発生した場合は即座に終了
					_ = pw.CloseWithError(writeErr)
					return
				}
			}
		}
	}()

	// 4. 既存の Usecase を呼び出す
	// ストリームから流れてくる pr (io.Reader) をそのまま domain.File に渡す
	f, err := domain.NewFile(meta.Filename, meta.ExpectedSize, pr)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "invalid metadata: %v", err)
	}

	// 既存のロジックを再利用（DIPの恩恵）
	err = h.interactor.UploadMultipleParallel(stream.Context(), []*domain.File{f})
	if err != nil {
		// ✨ エラーの詳細ログ記録
		// クライアントには詳細を隠しつつ、内部では原因を特定可能にする
		slog.Error("parallel upload failed in gRPC handler",
			"filename", meta.Filename,
			"error", err,
		)
		// HTTPの500に相当する gRPC Internal エラーを返す
		return status.Errorf(codes.Internal, "upload failed: %v", err)
	}

	// 成功ログ（オプション：あまりに頻繁なら Info より Debug でも可）
	slog.Info("gRPC upload completed successfully", "filename", meta.Filename)

	// ✨ 最終的な戻り値：成功時はクライアントにステータスを返して閉じる
	return stream.SendAndClose(&filepb.UploadFileResponse{
		Status:  "success",
		Message: "File uploaded successfully via gRPC stream",
	})
}
