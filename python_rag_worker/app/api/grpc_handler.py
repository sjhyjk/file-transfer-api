# python_rag_worker/app/api/grpc_handler.py

import logging
import os
from collections.abc import AsyncIterator
from typing import Any

import grpc

import pb.file_pb2 as file_pb2
import pb.file_pb2_grpc as file_pb2_grpc
from core.config import settings
from core.constants import GRPC_STATUS_MAP
from core.logger import AppLoggerAdapter, extract_trace_id
from services.rag_service import RAGService

logger = logging.getLogger(__name__)


class FileServiceServicer(file_pb2_grpc.FileServiceServicer):
    """Go サーバーからの gRPC リクエストを待ち受けるハンドラー"""

    def __init__(self, rag_service: RAGService) -> None:
        # main.py で初期化された RAGService のインスタンスを保持
        self.rag_service = rag_service
        self._base_logger = logger

    # 💡 grpcの同期メソッド内でasync関数（run_pipeline）を安全に実行するための即時実行
    async def NotifyIngest(
        self, request: Any, context: grpc.aio.ServicerContext
    ) -> Any:
        """Go からのファイルアップロード完了通知を受け取り、
        RAGパイプラインを非同期駆動する
        """
        try:
            # 🚀 トレースIDの抽出 (gRPC メタデータから取得)
            # 💡 invocation_metadata() が None のケースを考慮し、型安全に辞書化
            raw_metadata = context.invocation_metadata()
            metadata = dict(raw_metadata) if raw_metadata is not None else {}

            # メタデータからトレースIDを抽出
            raw_trace_id = metadata.get("x-trace-id")
            trace_id = extract_trace_id(raw_trace_id)

            # 2. 💡 このgRPCリクエスト専用のアダプターロガーを生成！
            req_logger = AppLoggerAdapter(
                self._base_logger, component="gRPC Ingest", trace_id=trace_id
            )

            file_id = request.file_id
            file_name = request.file_name
            tenant_id = request.tenant_id

            if not file_name:
                req_logger.error("💥 Rejecting request: missing file_name")
                await context.abort(
                    grpc.StatusCode.INVALID_ARGUMENT, "file_name is required"
                )

            if not tenant_id:
                req_logger.error("💥 Rejecting request: missing tenant_id")
                await context.abort(
                    grpc.StatusCode.INVALID_ARGUMENT, "tenant_id is required"
                )

            # 💡 proto の enum (整数値) を文字列名に変換してログの視認性を死守
            # 🚀 共通マップを使って、安全に Go のドメイン文字列（小文字）へ翻訳
            status = GRPC_STATUS_MAP.get(request.status, "unknown")

            # 💡 CreatedAt (Timestamp型) を文字列に落としてログに出す
            created_at = "N/A"
            if request.HasField("created_at"):
                created_at = request.created_at.ToJsonString()

            req_logger.info(
                f"📥 Received event | "
                f"Tenant: {tenant_id} | File: {file_name} (ID: {file_id}) | "
                f"Status: {status} | CreatedAt: {created_at}"
            )

            # config から安全にルートパスを取得して結合
            full_path = os.path.join(settings.storage_root, file_name)

            # 🚀 状態管理からインジェクションされた RAGService をきれいに await
            # ※必要に応じて、status や file_id もサービスに渡せるよう拡張可能です
            result = await self.rag_service.run_pipeline(tenant_id, full_path)

            if result.get("status") == "success":
                return file_pb2.FileIngestResponse(
                    status="success",
                    message=(
                        f"Successfully processed RAG pipeline via gRPC. "
                        f"Created {result.get('chunks_count')} chunks."
                    ),
                )
            else:
                # context.abort を非同期で await 呼び出し（grpc.aio の標準仕様）
                req_logger.warning(
                    f"⚠️ Pipeline returned failure status: {result.get('message')}"
                )
                await context.abort(
                    grpc.StatusCode.INTERNAL,
                    result.get("message", "Pipeline failed"),
                )

        except Exception as e:
            req_logger.error(f"💥 Failed to process ingestion: {e}")
            # エラー時は gRPC のステータスコードを適切に変えて返却
            await context.abort(grpc.StatusCode.INTERNAL, str(e))

    async def UploadFile(
        self,
        request_iterator: AsyncIterator[file_pb2.UploadFileRequest],
        context: grpc.aio.ServicerContext,
    ) -> file_pb2.UploadFileResponse:
        """Python側では未実装のため、UNIMPLEMENTEDを返却"""
        await context.abort(
            grpc.StatusCode.UNIMPLEMENTED, "Method not implemented in Python worker"
        )
        # 💡 Mypyの [return] エラーを消すため、ダミーの戻り値（または例外）を明示します
        raise NotImplementedError("Method not implemented in Python worker")

    async def ListFiles(
        self,
        request: file_pb2.ListFilesRequest,
        context: grpc.aio.ServicerContext,
    ) -> file_pb2.ListFilesResponse:
        """Python側では未実装のため、UNIMPLEMENTEDを返却"""
        await context.abort(
            grpc.StatusCode.UNIMPLEMENTED, "Method not implemented in Python worker"
        )
        # 💡 Mypyの [return] エラーを消すため、ダミーの戻り値（または例外）を明示します
        raise NotImplementedError("Method not implemented in Python worker")
