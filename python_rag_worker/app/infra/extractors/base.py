# python_rag_worker/app/infra/extractors/base.py

from abc import ABC, abstractmethod

from langchain_core.documents import Document


class BaseExtractor(ABC):
    @abstractmethod
    def extract(self, file_path: str) -> list[Document]:
        pass
