# python_rag_worker/app/infra/extractors/pdf_extractor.py

from langchain_community.document_loaders import PyPDFLoader
from langchain_core.documents import Document

from .base import BaseExtractor


class PDFExtractor(BaseExtractor):
    def extract(self, file_path: str) -> list[Document]:
        loader = PyPDFLoader(file_path)
        return loader.load()
