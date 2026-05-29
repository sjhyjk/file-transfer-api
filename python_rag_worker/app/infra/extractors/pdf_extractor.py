# python_rag_worker/app/infra/extractors/pdf_extractor.py

from typing import cast

from langchain_community.document_loaders import PyPDFLoader
from langchain_core.documents import Document

from .base import BaseExtractor


class PDFExtractor(BaseExtractor):
    def extract(self, file_path: str) -> list[Document]:
        loader = PyPDFLoader(file_path)
        return cast(list[Document], loader.load())
