# --- Build Stage ---
FROM golang:1.24-bookworm AS builder

WORKDIR /app

# 依存関係を先にコピーしてキャッシュを効かせる
COPY go.mod go.sum ./
RUN go mod download

# ソースコード全体をコピー
COPY . .

# 静的リンクしたバイナリをビルド（CGOをオフにして軽量化）
RUN CGO_ENABLED=0 GOOS=linux go build -o main ./cmd/api/main.go

# --- Run Stage ---
# 実行用には最小限のイメージを使用
FROM gcr.io/distroless/static-debian12

WORKDIR /

# ビルドしたバイナリと認証ファイルをコピー
COPY --from=builder /app/main .
COPY --from=builder /app/gcp-key.json .

# 実行
CMD ["./main"]
