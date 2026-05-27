# python_rag_worker/app/infra/extractors/pptx_extractor.py

from .base import BaseExtractor
from langchain_community.document_loaders import UnstructuredPowerPointLoader

class PPTXExtractor(BaseExtractor):
    def extract(self, file_path: str):
        loader = UnstructuredPowerPointLoader(file_path)
        return loader.load()
