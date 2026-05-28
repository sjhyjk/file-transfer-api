# python_rag_worker/app/api/grpc_server.py

import logging
from typing import Any

import grpc
import pb.file_pb2_grpc as file_pb2_grpc

from api.grpc_handler import FileServiceServicer
from core.config import settings

logger = logging.getLogger("rag-worker")


class GRPCServerManager:
    """gRPC サーバーのライフサイクル（起動・停止）を専門に管理するコンポーネント"""

    def __init__(self, rag_service: Any) -> None:
        self.rag_service = rag_service
        self._server: Any = None

    async def start(self) -> None:
        """非同期 gRPC サーバーを初期化してポートをバインド、起動する"""
        # 🚀 grpc.aio を使用して完全非同期なサーバーインスタンスを生成
        self._server = grpc.aio.server()

        # ハンドラー（サービサー）に共通の RAGService を注入してアタッチ
        file_pb2_grpc.add_FileServiceServicer_to_server(
            FileServiceServicer(rag_service=self.rag_service), self._server
        )  # type: ignore

        listen_addr = f"0.0.0.0:{settings.grpc_port}"
        self._server.add_insecure_port(listen_addr)

        logger.info(f"📡 gRPC Server is listening on {listen_addr}")
        await self._server.start()

    async def stop(self, grace: int = 5) -> None:
        """稼働中の gRPC サーバーを安全にシャットダウンする（Graceful Shutdown）"""
        if self._server:
            logger.info("Stopping Async gRPC server...")
            await self._server.stop(grace=grace)
            logger.info("gRPC server stopped safely.")
