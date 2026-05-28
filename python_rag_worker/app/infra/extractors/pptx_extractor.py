# python_rag_worker/app/infra/extractors/pptx_extractor.py

from langchain_community.document_loaders import UnstructuredPowerPointLoader
from langchain_core.documents import Document

from .base import BaseExtractor


class PPTXExtractor(BaseExtractor):
    def extract(self, file_path: str) -> list[Document]:
        loader = UnstructuredPowerPointLoader(file_path)
        return loader.load()
