# python_rag_worker/app/api/http_handler.py

import logging
import os
from typing import Any

from fastapi import APIRouter, HTTPException, Request

from core.config import settings
from core.logger import AppLoggerAdapter, extract_trace_id
from services.rag_service import RAGService

logger = logging.getLogger(__name__)


class HTTPHandler:
    """HTTP 経由のリクエストを待ち受け、RAGパイプラインへルーティングするハンドラー層"""

    def __init__(self, rag_service: RAGService) -> None:
        # main.py で初期化された RAGService のインスタンスを保持
        self.rag_service = rag_service
        self.router = APIRouter()
        self._base_logger = logger
        self._setup_routes()

    def _setup_routes(self) -> None:
        """ルーターのエンドポイントを登録"""
        self.router.add_api_route(
            "/health", self.health, methods=["GET"], tags=["Infrastructure"]
        )
        self.router.add_api_route(
            "/ingest", self.notify_ingest, methods=["POST"], tags=["Pipeline"]
        )

    def health(self) -> dict[str, str]:
        """インフラ監視用のヘルスチェックエンドポイント"""
        return {"status": "ok"}

    async def notify_ingest(self, request: Request) -> Any:
        """HTTP 経由のファイルアップロード完了通知を受付"""
        try:
            # 1. 🚀 gRPC と同様にヘッダーからトレースIDを抽出（無ければ生成）
            raw_trace_id = request.headers.get("x-trace-id")
            trace_id = extract_trace_id(raw_trace_id)

            # 2. 💡 このHTTPリクエスト専用のアダプターロガーを生成
            req_logger = AppLoggerAdapter(
                self._base_logger, component="HTTP Ingest", trace_id=trace_id
            )

            # JSONペイロードの解析
            payload = await request.json()

            file_id = payload.get("file_id")
            file_name = payload.get("file_name")
            tenant_id = payload.get("tenant_id")
            # 💡 Go側が string(meta.Status) で送る文字列
            status = payload.get("status")
            # 💡 RFC3339 形式の文字列
            created_at = payload.get("created_at")

            # 3. 🛡️ バリデーション
            if not file_name:
                raise HTTPException(
                    status_code=400, detail="💥 Rejecting request: missing file_name"
                )
            if not tenant_id:
                raise HTTPException(
                    status_code=400, detail="💥 Rejecting request: missing tenant_id"
                )

            # 4. 📝 統一されたきれいなインフォログ
            req_logger.info(
                f"📥 Received event | "
                f"Tenant: {tenant_id} | File: {file_name} (ID: {file_id}) | "
                f"Status: {status} | CreatedAt: {created_at}"
            )

            # 🚀 config からストレージのルートパスを取得
            full_path = os.path.join(settings.storage_root, file_name)

            # RAGパイプライン（抽出・分割）の実行
            result = await self.rag_service.run_pipeline(tenant_id, full_path)

            if result.get("status") == "success":
                return {
                    "status": "success",
                    "message": (
                        f"Successfully processed RAG pipeline via HTTP. "
                        f"Created {result.get('chunks_count')} chunks."
                    ),
                }
            else:
                req_logger.warning(
                    f"⚠️ Pipeline returned failure status: {result.get('message')}"
                )
                raise HTTPException(
                    status_code=500, detail=result.get("message", "Pipeline failed")
                )

        except HTTPException as he:
            raise he
        except Exception as e:
            req_logger.error(f"💥 Failed to process ingestion: {e}")
            raise HTTPException(status_code=500, detail=str(e)) from e
