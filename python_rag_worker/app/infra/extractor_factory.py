# python_rag_worker/app/infra/extractor_factory.py

import os
from .extractors.pdf_extractor import PDFExtractor
from .extractors.excel_extractor import ExcelExtractor
from .extractors.word_extractor import WordExtractor
from .extractors.pptx_extractor import PPTXExtractor
from .extractors.text_extractor import TextExtractor

class ExtractorFactory:
    _mapping = {
        ".pdf": PDFExtractor,
        ".xlsx": ExcelExtractor,
        ".xls": ExcelExtractor,
        ".docx": WordExtractor,
        ".pptx": PPTXExtractor,
        ".txt": TextExtractor,
    }

    @classmethod
    def get_extractor(cls, file_path: str):
        ext = os.path.splitext(file_path)[1].lower()
        extractor_class = cls._mapping.get(ext)
        
        if not extractor_class:
            raise ValueError(f"❌ Unsupported file extension: {ext}")
            
        return extractor_class()
