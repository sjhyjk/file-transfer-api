# python_rag_worker/app/infra/extractors/base.py

from abc import ABC, abstractmethod

class BaseExtractor(ABC):
    @abstractmethod
    def extract(self, file_path: str):
        pass
