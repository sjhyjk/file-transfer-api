# python_rag_worker/app/infra/extractors/word_extractor.py

from typing import cast
from langchain_community.document_loaders import Docx2txtLoader
from langchain_core.documents import Document

from .base import BaseExtractor


class WordExtractor(BaseExtractor):
    def extract(self, file_path: str) -> list[Document]:
        loader = Docx2txtLoader(file_path)
        return cast(list[Document], loader.load())
