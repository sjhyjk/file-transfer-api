# python_rag_worker/app/infra/extractors/pdf_extractor.py

from .base import BaseExtractor
from langchain_community.document_loaders import PyPDFLoader

class PDFExtractor(BaseExtractor):
    def extract(self, file_path: str):
        loader = PyPDFLoader(file_path)
        return loader.load()
