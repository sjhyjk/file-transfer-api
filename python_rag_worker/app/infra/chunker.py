# python_rag_worker/app/infra/chunker.py

import re

from langchain_core.documents import Document
from langchain_text_splitters import RecursiveCharacterTextSplitter


class Chunker:
    def __init__(self, chunk_size: int = 500, chunk_overlap: int = 50) -> None:
        self.splitter = RecursiveCharacterTextSplitter(
            chunk_size=chunk_size,
            chunk_overlap=chunk_overlap,
            separators=["\n\n", "\n", "。", "、", " ", ""]
        )

    def split(self, docs: list[Document]) -> list[Document]:
        # 💡 分割前に各ドキュメントのテキストをクリーニング
        for doc in docs:
            # 1. 連続する改行・空白を 1 つの改行に集約
            doc.page_content = re.sub(r'\n\s*\n', '\n', doc.page_content)
            # 2. 文頭・文末の余計な空白を削除
            doc.page_content = doc.page_content.strip()

        return self.splitter.split_documents(docs)
