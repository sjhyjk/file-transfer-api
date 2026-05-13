# python_rag_mock/main.py

from fastapi import FastAPI, Request
import logging

app = FastAPI()
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger("rag-mock")

@app.get("/health")
def health():
    return {"status": "ok"}

@app.post("/ingest")
async def ingest_file(request: Request):
    # Go 側の http_notifier から送られてくる JSON を受け取る
    payload = await request.json()

    # Go の http_notifier.go で定義したキーを取り出す
    file_id = payload.get("file_id")
    file_name = payload.get("file_name")
    
    logger.info("🚀 Received notification from Go API!")
    logger.info(f"📥 Goから通知を受信: ID={file_id}, Name={file_name}")
    
    return {
        "status": "received",
        "file_id": file_id,
        "message": "Start vector indexing..."
    }
