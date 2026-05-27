# python_rag_worker/app/core/config.py

import os
from dataclasses import dataclass

@dataclass(frozen=True)
class Settings:
    """アプリケーション全体の設定を一元管理するデータクラス"""
    
    # サーバー動作設定
    http_port: int = int(os.getenv("PYTHON_HTTP_PORT", "8000"))
    grpc_port: int = int(os.getenv("PYTHON_GRPC_PORT", "50051"))
    
    # ストレージ設定 (Go側の LOCAL_STORAGE_PATH と同期)
    storage_root: str = os.getenv("STORAGE_ROOT", "/app/storage")

# シングルトンとしてエクスポート
settings = Settings()
