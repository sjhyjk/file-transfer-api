# python_rag_worker/app/main_grpc.py

import asyncio
import logging
import signal
from typing import Any

import grpc

import pb.file_pb2_grpc as file_pb2_grpc
from api.grpc_handler import FileServiceServicer
from api.grpc_server import GRPCServerManager
from core.logger import init_logger
from services.rag_service import RAGService

# 1. ログの一元初期化
init_logger()
logger = logging.getLogger(__name__)


async def main() -> None:
    logger.info("🚀 [gRPC Server] Launching RAG Worker in [gRPC Mode]...")

    # 2. 共通コアロジックのインスタンス化
    rag_service_instance = RAGService(vector_store=None)
    grpc_handler = FileServiceServicer(rag_service=rag_service_instance)

    # 💡 登録関数(add_..._to_server)に具象サービサーを
    # 部分適用した内部定義関数を生成して渡す
    def register_fn(servicer: Any, server: grpc.aio.Server) -> None:
        file_pb2_grpc.add_FileServiceServicer_to_server(servicer, server)

    # 📡 [gRPC Server Component]
    # 3. gRPC サーバーマネージャーの初期化と起動
    server_manager = GRPCServerManager(register_fn=register_fn, servicer=grpc_handler)
    # 起動処理
    await server_manager.start()

    # 4. コンテナ終了シグナル（SIGTERM / SIGINT）への安全な対応
    stop_event = asyncio.Event()
    loop = asyncio.get_running_loop()

    def shutdown_signal_handler() -> None:
        logger.info("🚨 [gRPC Server] Shutdown signal received.")
        stop_event.set()

    # Linux (Docker) 環境でのクリーンアップ用シグナルを登録
    for sig in (signal.SIGINT, signal.SIGTERM):
        try:
            loop.add_signal_handler(sig, shutdown_signal_handler)
        except NotImplementedError:
            # Windows環境など、add_signal_handler が使えない場合のフォールバック
            pass

    # 5. 終了シグナルが飛んでくるまで、非同期でサーバーを維持（永続待機）
    logger.info(
        "💎 [gRPC Server] gRPC Worker is now running background loop. "
        "Press Ctrl+C to stop."
    )
    await stop_event.wait()

    # 6. 停止処理（Graceful Shutdown）
    await server_manager.stop(grace=5)


if __name__ == "__main__":
    # Python 3.14 の標準的な非同期エントリ実行
    asyncio.run(main())
