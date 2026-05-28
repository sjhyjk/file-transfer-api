# python_rag_worker/app/main_http.py

from collections.abc import AsyncGenerator
from contextlib import asynccontextmanager

from fastapi import FastAPI

from api.http_handler import router as http_router
from core.logger import init_logger
from services.rag_service import RAGService

# 1. ログの一元初期化（最優先で実行して、以降の全ログの見た目を統一する）
init_logger()


# 🚀 HTTPアプリ専用の lifespan ハンドラー
@asynccontextmanager
async def lifespan(app_instance: FastAPI) -> AsyncGenerator[None]:
    # 起動時の処理（DB接続初期化などが必要になればここに）
    # logger.info("HTTP Worker started.")

    yield  # ─── ここで FastAPI の実行フェーズ（リクエスト受付）へ移行 ───

    # 終了時の処理（クリーンアップなどが必要になればここに）
    # logger.info("HTTP Worker stopped.")


# 2. HTTP 専用アプリとして FastAPI を構成
app = FastAPI(
    title="RAG Worker [HTTP Mode]",
    description="REST API interface for RAG pipeline ingestion",
    lifespan=lifespan,  # 💡 ライフサイクルを登録
)

# 3. 依存関係（DI）のセットアップと状態管理への登録
# 将来的に pgvector などの具象クラスを渡す場合も、ここから注入（DI）します
# ルーターにサービスを紐付け（FastAPIの状態管理を利用）
rag_service_instance = RAGService(vector_store=None)
app.state.rag_service = rag_service_instance

# 4. HTTP ハンドラー（ルーター）の登録
app.include_router(http_router)
