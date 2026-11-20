# ==========================================
# 1. ツールビルドステージ（Goのツールをビルド）
# ==========================================
FROM golang:1.26-alpine AS tools-builder
RUN go install github.com/fe3dback/go-arch-lint@latest

# ==========================================
# 2. 統合静的解析ステージ（最終イメージ：Pythonベース）
# ==========================================
FROM python:3.14-alpine

# Goステージから 'go-arch-lint' のバイナリだけをコピー
COPY --from=tools-builder /go/bin/go-arch-lint /usr/local/bin/go-arch-lint

# ビルドに必要な最小限のパッケージと Git を入れる
RUN apk add --no-cache go build-base git

WORKDIR /app

# 開発環境用のツール定義をコピーして直接インストール
COPY internal/tools/analysis/requirements-dev.txt .
RUN pip install --no-cache-dir -r requirements-dev.txt
