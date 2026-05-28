# python_rag_worker/app/api/grpc_handler.py

import logging
import os
from typing import Any

import grpc
import pb.file_pb2 as file_pb2
import pb.file_pb2_grpc as file_pb2_grpc

from core.config import settings

logger = logging.getLogger("rag-worker")


class FileServiceServicer(file_pb2_grpc.FileServiceServicer):
    """Go サーバーからの gRPC リクエストを待ち受けるハンドラー"""

    def __init__(self, rag_service: Any) -> None:
        # main.py で初期化された RAGService のインスタンスを保持
        self.rag_service = rag_service

    # 💡 grpcの同期メソッド内でasync関数（run_pipeline）を安全に実行するための即時実行
    async def NotifyIngest(
        self, request: Any, context: grpc.aio.ServicerContext
    ) -> Any:
        """Go からのファイルアップロード完了通知を受け取り、
        RAGパイプラインを非同期駆動する
        """
        try:
            # 🚀 トレースIDの抽出 (gRPC メタデータから取得)
            metadata = dict(context.invocation_metadata())

            file_id = request.file_id
            file_name = request.file_name
            tenant_id = request.tenant_id
            if not tenant_id:
                logger.error("💥 [gRPC Ingest] Rejecting request: missing tenant_id")
                await context.abort(
                    grpc.StatusCode.INVALID_ARGUMENT, "tenant_id is required"
                )

            # トレースIDも、無ければその場で新規発行（uuid等）して、追跡性を死守する
            import uuid

            trace_id = metadata.get("x-trace-id", f"generated-{uuid.uuid4()}")

            logger.info(
                f"📥 [gRPC NotifyIngest] Received event | "
                f"TraceID: {trace_id} | Tenant: {tenant_id} | "
                f"File: {file_name} (ID: {file_id})"
            )

            # config から安全にルートパスを取得して結合
            full_path = os.path.join(settings.storage_root, file_name)

            # 🚀 状態管理からインジェクションされた RAGService をきれいに await
            result = await self.rag_service.run_pipeline(tenant_id, full_path)

            if result.get("status") == "success":
                return file_pb2.FileIngestResponse(  # type: ignore
                    status="success",
                    message=(
                        f"Successfully processed RAG pipeline via gRPC. "
                        f"Created {result.get('chunks_count')} chunks."
                    ),
                )
            else:
                # context.abort を非同期で await 呼び出し（grpc.aio の標準仕様）
                logger.warning(
                    f"⚠️ [gRPC Ingest] Pipeline returned failure status: "
                    f"{result.get('message')}"
                )
                await context.abort(
                    grpc.StatusCode.INTERNAL,
                    result.get("message", "Pipeline failed"),
                )

        except Exception as e:
            logger.error(f"💥 [gRPC NotifyIngest] Unexpected error: {e}")
            # エラー時は gRPC のステータスコードを適切に変えて返却
            await context.abort(grpc.StatusCode.INTERNAL, str(e))
