# python_rag_worker/app/main_grpc.py

import asyncio
import logging
import signal

from api.grpc_server import GRPCServerManager
from core.logger import init_logger
from services.rag_service import RAGService


async def main() -> None:
    # 1. ログの一元初期化
    init_logger()
    logger = logging.getLogger("rag-worker")
    logger.info("🚀 Launching RAG Worker in [gRPC Mode]...")

    # 2. 共通コアロジックのインスタンス化
    rag_service_instance = RAGService(vector_store=None)

    # 📡 [gRPC Server Component]
    # 3. gRPC サーバーマネージャーの初期化と起動
    server_manager = GRPCServerManager(rag_service=rag_service_instance)
    # 起動処理
    await server_manager.start()

    # 4. コンテナ終了シグナル（SIGTERM / SIGINT）への安全な対応
    stop_event = asyncio.Event()
    loop = asyncio.get_running_loop()

    def shutdown_signal_handler() -> None:
        logger.info("🚨 Shutdown signal received.")
        stop_event.set()

    # Linux (Docker) 環境でのクリーンアップ用シグナルを登録
    for sig in (signal.SIGINT, signal.SIGTERM):
        try:
            loop.add_signal_handler(sig, shutdown_signal_handler)
        except NotImplementedError:
            # Windows環境など、add_signal_handler が使えない場合のフォールバック
            pass

    # 5. 終了シグナルが飛んでくるまで、非同期でサーバーを維持（永続待機）
    logger.info("💎 gRPC Worker is now running background loop. Press Ctrl+C to stop.")
    await stop_event.wait()

    # 6. 停止処理（Graceful Shutdown）
    await server_manager.stop(grace=5)


if __name__ == "__main__":
    # Python 3.14 の標準的な非同期エントリ実行
    asyncio.run(main())
