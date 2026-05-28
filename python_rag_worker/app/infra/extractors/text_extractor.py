# python_rag_worker/app/infra/extractors/text_extractor.py

from langchain_community.document_loaders import TextLoader
from langchain_core.documents import Document

from .base import BaseExtractor


class TextExtractor(BaseExtractor):
    def extract(self, file_path: str) -> list[Document]:
        # 日本語環境を考慮し、encoding="utf-8" を指定（必要に応じて調整）
        loader = TextLoader(file_path, encoding="utf-8")
        return loader.load()
