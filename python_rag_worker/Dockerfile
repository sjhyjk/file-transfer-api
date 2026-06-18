# ==========================================
# 1. Builder Stage: 依存関係のコンパイル環境
# ==========================================

# 軽量なPythonイメージ
FROM python:3.14-slim AS builder

WORKDIR /app

ENV PYTHONUNBUFFERED=1
ENV PYTHONPATH=/app

# 1. 依存するシステムパッケージのインストール
# grpcio 等のビルドに必要な gcc や python3-dev を追加
RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    gcc \
    libc6-dev \
    python3-dev \
    && rm -rf /var/lib/apt/lists/*

# 2. requirements.txt のコピーとインストール

# 👑 requirements をコピーする前に、pip 自体のアップグレードを先に済ませてキャッシュ化する
RUN pip install --no-cache-dir --upgrade pip

COPY requirements.txt .

RUN pip install --no-cache-dir --user -r requirements.txt

# ==========================================
# 2. Runtime Stage: 最小特権実行環境
# ==========================================
FROM python:3.14-slim AS runtime

# タイムゾーン等の設定（ログを見やすくするため）
ENV TZ=Asia/Tokyo
ENV PYTHONDONTWRITEBYTECODE=1
ENV PYTHONUNBUFFERED=1
ENV PYTHONPATH=/app
ENV PATH=/root/.local/bin:$PATH

# 💡 本番イメージには gcc も build-essential も一切入れない（アタックサーフェスの最小化）
# タイムゾーンデータの最小構成のみを保証
RUN apt-get update && apt-get install -y --no-install-recommends \
    tzdata \
    && rm -rf /var/lib/apt/lists/*

# Builderステージからコンパイル済みの安全なライブラリ群のみをコピー
COPY --from=builder /root/.local /root/.local

# 💡 3. ディレクトリ構造を保ったままソースコードをコピー
COPY app/ /app/

WORKDIR /app

# ポートの明示 (FastAPI: 8081, gRPC: 50051)
EXPOSE 8081
EXPOSE 50051

# FastAPIを起動 (ポート8081)
# --host 0.0.0.0 はコンテナ外からのアクセスを許可するために必須
# デフォルトのエントリーポイント（Compose側で上書き可能）
CMD ["uvicorn", "main_http:app", "--host", "0.0.0.0", "--port", "8081"]
