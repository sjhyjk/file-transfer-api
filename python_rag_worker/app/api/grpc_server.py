# python_rag_worker/app/api/grpc_server.py

import asyncio
import logging
from collections.abc import Callable
from typing import Any

import grpc

from core.config import settings
from core.logger import AppLoggerAdapter

logger = logging.getLogger(__name__)


class GRPCServerManager:
    """gRPC サーバーのライフサイクル（起動・停止）を専門に管理するコンポーネント"""

    def __init__(
        self, register_fn: Callable[[Any, grpc.aio.Server], None], servicer: Any
    ) -> None:
        """
        Args:
            register_fn: pb2_grpc.add_XServicer_to_server などの登録関数
            servicer: 初期化済みのサービサーインスタンス (例: FileServiceServicer)
        """
        self._register_fn = register_fn
        self._servicer = servicer
        self._server: grpc.aio.Server | None = None

        self.logger = AppLoggerAdapter(logger, component="gRPC Server")

    async def start(self) -> None:
        """非同期 gRPC サーバーを初期化してポートをバインド、起動する"""
        try:
            # 🚀 grpc.aio を使用して完全非同期なサーバーインスタンスを生成
            self._server = grpc.aio.server()

            # 外から渡された登録関数を使って、サービサーをアタッチ
            self._register_fn(self._servicer, self._server)

            listen_addr = f"0.0.0.0:{settings.grpc_port}"
            self._server.add_insecure_port(listen_addr)

            self.logger.info(f"📡 listening on {listen_addr}")
            await self._server.start()

        except Exception as e:
            self.logger.error(f"💥 Critical error during server startup: {e}")
            raise e

    async def stop(self, grace: int = 5) -> None:
        """稼働中の gRPC サーバーを安全にシャットダウンする（Graceful Shutdown）"""
        if not self._server:
            self.logger.warning("⚠️ Server is not running.")
            return

        self.logger.info("👋 Stopping Async gRPC server...")
        try:
            # gRPC aio の仕様に合わせて指定秒数の猶予を持ってクローズ
            await self._server.stop(grace=grace)
            self.logger.info("✅ gRPC server stopped safely.")
        except asyncio.CancelledError:
            self.logger.warning("⚠️ Server shutdown task was cancelled abruptly.")
        except Exception as e:
            self.logger.error(f"🔥 Unexpected error during server shutdown: {e}")
