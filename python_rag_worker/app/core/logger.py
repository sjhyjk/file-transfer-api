# python_rag_worker/app/core/logger.py

import logging
import sys


def init_logger() -> None:
    """アプリケーション全体のログ設定を初期化する"""

    # ルートロガーのフォーマット設定
    log_format = '%(asctime)s [%(levelname)s] %(name)s: %(message)s'

    logging.basicConfig(
        level=logging.INFO,
        format=log_format,
        handlers=[
            # sys.stdout へ出力することで、コンテナ環境でのログ収集
            # （FluentbitやCloud Loggingなど）と親和性を高める
            logging.StreamHandler(sys.stdout)
        ]
    )

    # 起動時の確認用ログ
    logger = logging.getLogger("rag-worker")
    logger.info("✨ Structured logging initialized successfully.")
