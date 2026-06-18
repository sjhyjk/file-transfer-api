# python_rag_worker/app/main_pubsub.py

import asyncio
import logging
import signal

from api.pubsub_handler import PubSubHandler
from api.pubsub_server import PubSubServerManager
from core.logger import init_logger
from services.rag_service import RAGService

init_logger()
logger = logging.getLogger(__name__)


async def main() -> None:
    logger.info("🚀 [Pub/Sub Server] Launching RAG Worker in [Pub/Sub Mode]...")

    # 1. 依存関係の注入 (DI)
    rag_service_instance = RAGService(vector_store=None)
    pubsub_handler = PubSubHandler(rag_service=rag_service_instance)

    # 2. サーバーマネージャーの初期化と起動
    server_manager = PubSubServerManager(processor=pubsub_handler)
    await server_manager.start()

    # 3. 終了シグナル監視機構
    stop_event = asyncio.Event()
    loop = asyncio.get_running_loop()

    def shutdown_signal_handler() -> None:
        logger.info("🚨 [Pub/Sub Server] Shutdown signal received.")
        stop_event.set()

    for sig in (signal.SIGINT, signal.SIGTERM):
        try:
            loop.add_signal_handler(sig, shutdown_signal_handler)
        except NotImplementedError:
            pass

    logger.info(
        "💎 [Pub/Sub Server] Pub/Sub Worker is now running background loop. "
        "Press Ctrl+C to stop."
    )

    # 終了シグナルがトリガーされるまで、メインループを非同期でクリーンに待機
    await stop_event.wait()

    # 4. 安全な停止（Graceful Shutdown）
    await server_manager.stop(grace=5)


if __name__ == "__main__":
    asyncio.run(main())
