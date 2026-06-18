# python_rag_worker/app/core/logger.py

import logging
import sys
import uuid
from collections.abc import MutableMapping
from typing import Any


class DefaultFieldsFilter(logging.Filter):
    """
    外部ライブラリ（Uvicorn, gRPC等）が独自のログを出力した際、
    フォーマットに含まれる component や trace_id が無くて
    KeyError になるのを防ぐフィルター。
    """

    def filter(self, record: logging.LogRecord) -> bool:
        if not hasattr(record, "component"):
            record.component = "System"
        if not hasattr(record, "trace_id"):
            record.trace_id = "system"
        return True


def init_logger() -> None:
    """アプリケーション全体のログ設定を初期化する"""

    # ルートロガーのフォーマット設定
    log_format = (
        "%(asctime)s [%(levelname)s] [%(component)s] "
        "[Trace: %(trace_id)s] %(name)s: %(message)s"
    )

    # ハンドラーの生成
    stream_handler = logging.StreamHandler(sys.stdout)

    # 💡 フィルターをアタッチして、不足している属性を安全に補完
    stream_handler.addFilter(DefaultFieldsFilter())

    logging.basicConfig(
        level=logging.INFO,
        format=log_format,
        handlers=[
            # sys.stdout へ出力することで、コンテナ環境でのログ収集
            # （FluentbitやCloud Loggingなど）と親和性を高める
            stream_handler
        ],
    )

    # 起動時の確認用ログ
    logger = logging.getLogger("rag-worker")
    logger.info(
        "✨ Structured logging initialized successfully.",
        extra={"component": "Core", "trace_id": "system"},
    )


# トレースIDも、無ければその場で新規発行（uuid等）して、追跡性を死守する
def extract_trace_id(raw_trace_id: str | bytes | None) -> str:
    """型（None, bytes, str）を問わず、安全に文字列のトレースIDを抽出・生成する"""
    if raw_trace_id is None:
        return f"generated-{uuid.uuid4().hex[:8]}"
    if isinstance(raw_trace_id, bytes):
        try:
            return raw_trace_id.decode("utf-8")
        except Exception:
            return f"gen-{uuid.uuid4().hex[:8]}"
    else:
        # 既に通常の文字列であればそのまま採用
        return str(raw_trace_id).strip()


class AppLoggerAdapter(logging.LoggerAdapter):
    """
    コンポーネント名とトレースIDをログに自動埋め込みするためのカスタムアダプター。
    これを通すことで、各層の logger.info("Message") に自動でプレフィックスが付与される。
    """

    def __init__(
        self, logger: logging.Logger, component: str, trace_id: str = "system"
    ) -> None:
        super().__init__(logger, {"component": component, "trace_id": trace_id})

    def process(
        self, msg: Any, kwargs: MutableMapping[str, Any]
    ) -> tuple[Any, MutableMapping[str, Any]]:
        # extra 辞書をマージして logging システムに渡す
        extra = dict(self.extra) if self.extra is not None else {}

        if "extra" in kwargs and kwargs["extra"] is not None:
            extra.update(kwargs["extra"])

        kwargs["extra"] = extra
        return msg, kwargs
