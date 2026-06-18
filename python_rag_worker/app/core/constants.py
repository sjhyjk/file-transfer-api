# python_rag_worker/app/core/constants.py

import pb.file_pb2 as file_pb2

# 💡 gRPC の Enum 値から、Goのドメイン定義（文字列）へ安全に翻訳するマップ
GRPC_STATUS_MAP = {
    file_pb2.TRANSFER_STATUS_PENDING: "pending",
    file_pb2.TRANSFER_STATUS_PROCESSING: "processing",
    file_pb2.TRANSFER_STATUS_COMPLETED: "completed",
    file_pb2.TRANSFER_STATUS_FAILED: "failed",
}
