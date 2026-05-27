# python_rag_worker/app/infra/extractors/excel_extractor.py

from .base import BaseExtractor
from langchain_core.documents import Document
from langchain_community.document_loaders import UnstructuredExcelLoader

class ExcelExtractor(BaseExtractor):
    def extract(self, file_path: str):
        # mode="elements" にするとセル単位になりますが、まずは全体を1つに
        loader = UnstructuredExcelLoader(file_path, mode="single")
        return loader.load()
