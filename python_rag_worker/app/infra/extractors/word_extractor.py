# python_rag_worker/app/infra/extractors/word_extractor.py

from .base import BaseExtractor
from langchain_community.document_loaders import Docx2txtLoader

class WordExtractor(BaseExtractor):
    def extract(self, file_path: str):
        loader = Docx2txtLoader(file_path)
        return loader.load()
