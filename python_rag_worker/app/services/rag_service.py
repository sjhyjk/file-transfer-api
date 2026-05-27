# python_rag_worker/app/services/rag_service.py

from infra.extractor_factory import ExtractorFactory
from infra.chunker import Chunker
import logging

logger = logging.getLogger("rag-worker")

class RAGService:
    def __init__(self, vector_store=None):
        # 💡 今は None だが、将来的に pgvector 等の具象クラスを注入する設計であることを示す
        self.vector_store = vector_store

        # チャンカーは形式を問わず共通の設定を利用
        self.chunker = Chunker()

    async def run_pipeline(self, tenant_id: str, file_path: str):
        """
        RAG パイプラインのオーケストレーション
        1. Factory を通じて適切な抽出器を取得
        2. テキスト抽出 (PDF, Excel, Word, PPTX 全対応)
        3. チャンク分割
        4. (将来実装) ベクトル化 & DB登録
        """

        try:
            logger.info(f"🚀 [Tenant:{tenant_id}] Starting pipeline for: {file_path}")

            # 1. 工場(Factory)から適切な抽出器を自動選択
            # 内部で拡張子を判定し、PDFExtractor や ExcelExtractor 等を返す
            extractor = ExtractorFactory.get_extractor(file_path)
            
            # 2. テキスト抽出
            # 各形式の Extractor は共通の extract() メソッドを持つため、
            # サービス層は中身が何かを気にせず呼び出せる（ポリモーフィズム）
            docs = extractor.extract(file_path)
            
            # 3. チャンク分割
            chunks = self.chunker.split(docs)
            
            logger.info(f"✅ [Tenant:{tenant_id}] Successfully created {len(chunks)} chunks.")
        
            # 💡 TODO: Phase 2 でベクトル化とDB登録を実装
            # self.vector_store.upsert(chunks, tenant_id=tenant_id)
            for i, chunk in enumerate(chunks[:2]):
                print(f"   [🚧 TODO: Vectorize] Chunk {i}: {chunk.page_content[:50]}...")

            # 4. 返却 (将来的にベクトルDBのメタデータなどを含める)
            return {
                "status": "success",
                "tenant_id": tenant_id,
                "file_path": file_path,
                "chunks_count": len(chunks)
            }

        except ValueError as e:
            logger.error(f"❌ [Tenant:{tenant_id}] Unsupported file error: {e}")
            return {"status": "error", "message": str(e)}
            
        except Exception as e:
            logger.error(f"🔥 [Tenant:{tenant_id}] Unexpected error: {e}")
            return {"status": "error", "message": "Internal pipeline error"}
