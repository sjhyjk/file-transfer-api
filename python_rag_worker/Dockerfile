# 軽量なPythonイメージ
FROM python:3.14-slim

WORKDIR /app

# タイムゾーン等の設定（ログを見やすくするため）
ENV TZ=Asia/Tokyo
ENV PYTHONUNBUFFERED=1

# 1. 依存するシステムパッケージのインストール
# grpcio 等のビルドに必要な gcc や python3-dev を追加
RUN apt-get update && apt-get install -y --no-install-recommends \
    gcc \
    libc6-dev \
    python3-dev \
    && rm -rf /var/lib/apt/lists/*

# 2. requirements.txt のコピーとインストール
COPY requirements.txt .
RUN pip install --no-cache-dir --upgrade pip && \
    pip install --no-cache-dir -r requirements.txt

# ソースコードのコピー
COPY . .

# FastAPIを起動 (ポート8000)
# --host 0.0.0.0 はコンテナ外からのアクセスを許可するために必須
CMD ["uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8000"]
