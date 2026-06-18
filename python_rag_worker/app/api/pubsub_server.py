# python_rag_worker/app/api/pubsub_server.py

import asyncio
import logging
from collections.abc import AsyncGenerator
from typing import Any, Protocol

from google.api_core.exceptions import NotFound
from google.cloud import pubsub_v1
from google.pubsub_v1 import AcknowledgeRequest, StreamingPullRequest
from google.pubsub_v1.services.subscriber import SubscriberAsyncClient

from core.config import settings
from core.logger import AppLoggerAdapter

logger = logging.getLogger(__name__)


class PubSubMessageProcessor(Protocol):
    """サーバー層が依存するハンドラーのインターフェースを定義（逆転の原則）"""

    async def process_message(
        self,
        message: Any,
        subscriber_client: SubscriberAsyncClient,
        subscription_path: str,
        ack_id: str,
    ) -> None:
        """この型シグネチャをハンドラー側が実装していれば、どんな具象クラスでも受け付ける"""
        ...


class PubSubServerManager:
    """GCP Pub/Sub の非同期サブスクリプション・
    ライフサイクルを純粋に管理するサーバー層
    """

    def __init__(self, processor: PubSubMessageProcessor) -> None:
        """
        Args:
            processor: PubSubMessageProcessor インターフェースを満たす
            ハンドラーインスタンス
        """
        self.processor = processor
        # 🚀 完全非同期(Async)に対応したサブスクライダークライアントを生成
        self.subscriber = SubscriberAsyncClient()
        self.subscription_path = pubsub_v1.SubscriberClient.subscription_path(
            settings.gcp_project_id, settings.pubsub_subscription_id
        )

        self._main_task: asyncio.Task[None] | None = None
        self._stop_event = asyncio.Event()

        # 💡 各クラス専用のアダプターを生成（グローバルロガーからの脱却）
        # これにより self.logger.info() を呼ぶだけで自動的にプレフィックスが付きます
        self.logger = AppLoggerAdapter(logger, component="Pub/Sub Server")

    async def start(self) -> None:
        """Pub/Subの非同期ループをバックグラウンドタスクとして起動"""
        try:
            self._stop_event.clear()
            # メインループを asyncio.Task としてスケジュール
            self._main_task = asyncio.create_task(self._subscribe_loop())

            self.logger.info(f"📡 Ready to subscribe to: {self.subscription_path}")

            # 🚀 サブスクリプションの存在チェック ＆ セルフリトライロジック
            max_retries = 10
            retry_interval = 3  # 秒
            is_ready = False

            # subscriberコンテキストを事前に開いてチェックを実行
            for attempt in range(1, max_retries + 1):
                try:
                    # サブスクリプションが存在するかメタデータを取得してみる
                    await self.subscriber.get_subscription(
                        request={"subscription": self.subscription_path}
                    )
                    is_ready = True
                    self.logger.info(
                        f"✅ Subscription verified successfully (Attempt {attempt})."
                    )
                    break
                except NotFound:
                    if attempt == max_retries:
                        self.logger.critical(
                            f"🔥 Max retries ({max_retries}) reached. "
                            f"Subscription '{self.subscription_path}' "
                            f"does not exist in emulator."
                        )
                        raise

                    self.logger.warning(
                        f"⏳ Subscription not found "
                        f"(Attempt {attempt}/{max_retries}). "
                        f"Waiting {retry_interval}s "
                        f"for emulator initialization resources..."
                    )
                    # イベントループをブロックせずに指定秒数待機
                    await asyncio.sleep(retry_interval)

            # 💡 リソースの存在を確認後、メインループを asyncio.Task としてスケジュール
            if is_ready:
                self._main_task = asyncio.create_task(self._subscribe_loop())
                self.logger.info(f"📡 Ready to subscribe to: {self.subscription_path}")

        except Exception as e:
            self.logger.error(f"💥 Critical error during server startup: {e}")
            raise e

    async def _subscribe_loop(self) -> None:
        """スレッドを一切生成しない、純粋な asyncio ベースのストリーミングプルループ"""
        try:
            # 🚀 完全非同期コンテキストでのメッセージ受信ストリームを開く
            async with self.subscriber:
                # 💡 内部に非同期ジェネレータを定義し、
                # リクエストをストリームに流し続けられるようにする
                async def request_generator() -> AsyncGenerator[StreamingPullRequest]:
                    # 👑 1発目のメッセージ: stream_ack_deadline_seconds を含めて初期化
                    # （400エラー回避）
                    yield StreamingPullRequest(
                        subscription=self.subscription_path,
                        stream_ack_deadline_seconds=60,
                    )

                    # 🛑 2発目以降は yield（送信）を一切行いません。
                    # ただし、関数自体が終了したりフリーズするとストリームが死ぬため、
                    # 終了フラグが立つまで「何もせずただループを維持」します。
                    while not self._stop_event.is_set():
                        await asyncio.sleep(1)

                # 🚀 ジェネレータを requests に直接叩き込む
                stream = await self.subscriber.streaming_pull(
                    requests=request_generator()
                )

                async for response in stream:
                    if self._stop_event.is_set():
                        break

                    # 受信したメッセージ群を処理
                    for received_message in response.received_messages:
                        msg_id = received_message.message.message_id
                        self.logger.info(f"📩 Message received! Message ID: {msg_id}")

                        # 💡 サーバー層は単にメッセージをパケットとして
                        #  ハンドラーへ丸投げするだけ
                        # Ack/Nackのクライアント操作もハンドラー側に委託
                        asyncio.create_task(
                            self._safe_process(
                                message=received_message.message,
                                ack_id=received_message.ack_id,
                            )
                        )

        except asyncio.CancelledError:
            self.logger.warning("⚠️ Subscription loop task was cancelled abruptly.")
        except Exception as e:
            self.logger.error(f"🔥 Critical error inside Pub/Sub async loop: {e}")

    async def _safe_process(self, message: Any, ack_id: str) -> None:
        """個別メッセージの例外がメインループを壊さないためのラッパー"""
        try:
            await self.processor.process_message(
                message=message,
                subscriber_client=self.subscriber,
                subscription_path=self.subscription_path,
                ack_id=ack_id,
            )
        except Exception as msg_err:
            self.logger.error(f"💥 Handler raised unhandled exception: {msg_err}")

        finally:
            try:
                await self.subscriber.acknowledge(
                    request=AcknowledgeRequest(
                        subscription=self.subscription_path,
                        ack_ids=[ack_id],
                    )
                )
                self.logger.debug(
                    f"🧹 Message {message.message_id} acknowledged successfully."
                )
            except Exception as ack_err:
                self.logger.error(
                    f"🔥 Failed to send ACK for message {message.message_id}: {ack_err}"
                )

    async def stop(self, grace: int = 5) -> None:
        """サブスクライバーを安全に停止（Graceful Shutdown）"""
        if not self._main_task or self._main_task.done():
            self.logger.warning("⚠️ Subscriber loop is not running.")
            return

        self.logger.info("👋 Stopping subscriber server...")
        self._stop_event.set()
        self._main_task.cancel()

        try:
            # grace 秒だけ、タスクが綺麗に終了クリーンアップを終えるのを待つ
            await asyncio.wait_for(self._main_task, timeout=grace)
            self.logger.info("✅ Subscriber server stopped safely.")
        except TimeoutError:
            self.logger.warning(
                f"⚠️ Server shutdown timed out within {grace}s grace period."
            )
        except asyncio.CancelledError:
            self.logger.warning(
                f"⚠️ Server shutdown timed out within {grace}s grace period."
            )
        except Exception as e:
            # 💡 キャンセル以外の原因でメインタスクが落ちていた場合はエラー記録
            self.logger.error(f"🔥 Unexpected error during main task cancellation: {e}")
