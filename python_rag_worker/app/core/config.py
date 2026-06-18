# python_rag_worker/app/core/config.py

import os
from dataclasses import dataclass


@dataclass(frozen=True)
class Settings:
    """アプリケーション全体の設定を一元管理するデータクラス"""

    # サーバー動作設定
    http_port: int = int(os.getenv("HTTP_PORT", "8081"))
    grpc_port: int = int(os.getenv("GRPC_PORT", "50051"))

    # Pub/Sub 設定
    gcp_project_id: str = os.getenv("GCP_PROJECT_ID", "file-transfer-api-project")
    pubsub_subscription_id: str = os.getenv("PUBSUB_SUBSCRIPTION_ID", "file-ingest-sub")

    # ストレージ設定 (Go側の LOCAL_STORAGE_PATH と同期)
    storage_root: str = os.getenv("STORAGE_ROOT", "/app/storage")


# シングルトンとしてエクスポート
settings = Settings()
