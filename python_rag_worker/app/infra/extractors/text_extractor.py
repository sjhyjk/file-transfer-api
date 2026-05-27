# python_rag_worker/app/infra/extractors/text_extractor.py

from .base import BaseExtractor
from langchain_community.document_loaders import TextLoader

class TextExtractor(BaseExtractor):
    def extract(self, file_path: str):
        # 日本語環境を考慮し、encoding="utf-8" を指定（必要に応じて調整）
        loader = TextLoader(file_path, encoding="utf-8")
        return loader.load()
