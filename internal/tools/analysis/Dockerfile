# ==========================================
# 1. ツールビルドステージ（Goのツールを軽量にビルド）
# ==========================================
FROM golang:1.25-alpine AS tools-builder
RUN go install github.com/fe3dback/go-arch-lint@latest

# ==========================================
# 2. Python依存関係ビルドステージ
# ==========================================
FROM python:3.14-alpine AS python-builder
WORKDIR /build

# 開発環境用のツール定義をコピーしてインストール
COPY internal/tools/analysis/requirements-dev.txt .
RUN apk add --no-cache build-base python3-dev git && \
    pip install --no-cache-dir --target=/build/packages -r requirements-dev.txt

# ==========================================
# 3. 統合静的解析ステージ（最終イメージ）
# ==========================================
FROM golang:1.25-alpine

# 各ステージから成果物のみをクリーンにコピー
COPY --from=tools-builder /go/bin/go-arch-lint /usr/local/bin/go-arch-lint
COPY --from=python-builder /build/packages /usr/local/lib/python3.14/site-packages

# 実行に必要な最小限の Python ランタイムと Git だけを入れる
RUN apk add --no-cache python3 git

# Python パスを通す
ENV PYTHONPATH=/usr/local/lib/python3.14/site-packages
WORKDIR /app
