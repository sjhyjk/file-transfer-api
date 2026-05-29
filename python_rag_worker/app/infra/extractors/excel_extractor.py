# python_rag_worker/app/infra/extractors/excel_extractor.py

from typing import cast
from langchain_community.document_loaders import UnstructuredExcelLoader
from langchain_core.documents import Document

from .base import BaseExtractor


class ExcelExtractor(BaseExtractor):
    def extract(self, file_path: str) -> list[Document]:
        # mode="elements" にするとセル単位になりますが、まずは全体を1つに
        loader = UnstructuredExcelLoader(file_path, mode="single")
        return cast(list[Document], loader.load())
