# python_rag_worker/app/infra/extractors/pptx_extractor.py

from typing import cast
from langchain_community.document_loaders import UnstructuredPowerPointLoader
from langchain_core.documents import Document

from .base import BaseExtractor


class PPTXExtractor(BaseExtractor):
    def extract(self, file_path: str) -> list[Document]:
        loader = UnstructuredPowerPointLoader(file_path)
        return cast(list[Document], loader.load())
