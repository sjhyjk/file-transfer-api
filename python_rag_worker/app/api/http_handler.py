# python_rag_worker/app/api/http_handler.py

import logging
import os
from typing import Any

from fastapi import APIRouter, HTTPException, Request

from core.config import settings

logger = logging.getLogger("rag-worker")

router = APIRouter()


@router.get("/health", tags=["Infrastructure"])  # type: ignore[untyped-decorator]
def health() -> dict[str, str]:
    """インフラ監視用のヘルスチェックエンドポイント"""
    return {"status": "ok"}


@router.post("/ingest", tags=["Pipeline"])  # type: ignore[untyped-decorator]
async def ingest_notification(request: Request) -> Any:
    """HTTP 経由のファイルアップロード完了通知を受付"""
    try:
        payload = await request.json()

        file_id = payload.get("file_id")
        file_name = payload.get("file_name")
        tenant_id = payload.get("tenant_id")

        if not file_name:
            raise HTTPException(status_code=400, detail="file_name is required")
        if not tenant_id:
            raise HTTPException(status_code=400, detail="tenant_id is required")

        # 🚀 config からストレージのルートパスを取得
        full_path = os.path.join(settings.storage_root, file_name)

        # main.py で作ったインスタンスを呼び出す
        rag_service = request.app.state.rag_service

        logger.info(
            f"📥 [HTTP Ingest] Received event | "
            f"Tenant: {tenant_id} | File: {file_name} (ID: {file_id})"
        )

        # RAGパイプライン（抽出・分割）の実行
        result = await rag_service.run_pipeline(tenant_id, full_path)
        return result

    except HTTPException as he:
        raise he
    except Exception as e:
        logger.error(f"💥 [HTTP Ingest] Failed to process ingestion: {e}")
        raise HTTPException(status_code=500, detail=str(e)) from e
