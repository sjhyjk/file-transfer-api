# python_rag_worker/app/api/pubsub_handler.py

import json
import logging
import os
from typing import Any

from google.cloud import pubsub_v1

from core.config import settings
from core.logger import AppLoggerAdapter, extract_trace_id
from services.rag_service import RAGService

logger = logging.getLogger(__name__)


class PubSubHandler:
    """Pub/Subメッセージのパースとバリデーションを担当するハンドラー層"""

    def __init__(self, rag_service: RAGService) -> None:
        self.rag_service = rag_service
        # 💡 ベースとなるルートロガーだけインスタンス変数として保持しておく
        self._base_logger = logger

    async def process_message(
        self, message: Any, subscriber_client: Any, subscription_path: str, ack_id: str
    ) -> None:
        """ServerManager からメッセージパケットを引き受けるエントリーポイント"""
        # 1. トレースIDの抽出
        attributes = message.attributes if message.attributes else {}
        raw_trace_id = attributes.get("x-trace-id")
        trace_id = extract_trace_id(raw_trace_id)

        # 2. 💡 このリクエスト専用のアダプターロガーを生成！
        req_logger = AppLoggerAdapter(
            self._base_logger, component="Pub/Sub Ingest", trace_id=trace_id
        )

        # 3. 実際の処理メソッドへ、このロガーを託して処理を移譲する
        success = await self.handle_message_core(message, req_logger)

        # 4. 結果に応じた Ack / Nack 制御
        if success:
            await subscriber_client.acknowledge(
                request={"subscription": subscription_path, "ack_ids": [ack_id]}
            )
        else:
            await subscriber_client.modify_ack_deadline(
                request={
                    "subscription": subscription_path,
                    "ack_ids": [ack_id],
                    "ack_deadline_seconds": 0,
                }
            )

    async def handle_message_core(
        self, message: pubsub_v1.types.PubsubMessage, req_logger: AppLoggerAdapter
    ) -> bool:
        """Pub/Subスレッドから呼び出される同期コールバック"""
        try:
            # 1. メッセージのデコードとパース
            data = json.loads(message.data.decode("utf-8"))

            file_id = data.get("file_id")
            file_name = data.get("file_name")
            tenant_id = data.get("tenant_id")
            # 💡 文字列ステータス
            status = data.get("status")
            # 💡 RFC3339 形式の文字列
            created_at = data.get("created_at")

            # 2. バリデーション
            if not file_name:
                req_logger.error("💥 Rejecting request: missing file_name")
                return False

            if not tenant_id:
                req_logger.error("💥 Rejecting request: missing tenant_id")
                return False

            req_logger.info(
                f"📥 Received event | "
                f"Tenant: {tenant_id} | File: {file_name} (ID: {file_id}) | "
                f"Status: {status} | CreatedAt: {created_at}"
            )

            # 3. パス組み立て
            full_path = os.path.join(settings.storage_root, file_name)

            # 🚀 完全非同期なので、そのままストレートに await 可能！
            result = await self.rag_service.run_pipeline(tenant_id, full_path)

            if result.get("status") == "success":
                req_logger.info(
                    f"✅ RAG Pipeline finished successfully for: {file_name}"
                )
                # 処理成功 -> サーバー側で Ack 処理される
                return True
            else:
                req_logger.warning(
                    f"⚠️ Pipeline returned failure status: {result.get('message')}"
                )
                return False

        except Exception as e:
            req_logger.error(f"💥 Failed to process ingestion: {e}")
            return False
