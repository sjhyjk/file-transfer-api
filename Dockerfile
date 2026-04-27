# --- Build Stage ---
FROM golang:1.25-bookworm AS builder

WORKDIR /app

# 1. 依存関係を先にコピーしてキャッシュを効かせる
COPY go.mod go.sum ./
RUN go mod download

# 2. 全ソースコピー（migrations等も含む）
COPY . .

# 3. 静的リンクしたバイナリをビルド（CGOをオフにして軽量化）
RUN CGO_ENABLED=0 GOOS=linux go build -o main ./cmd/api/main.go

# 4. 書き込み用ディレクトリの事前作成（Distroless対策）
RUN mkdir -p /app/storage

# --- Run Stage ---
# 実行用には最小限のイメージを使用
FROM gcr.io/distroless/static-debian12

WORKDIR /

# ビルドしたバイナリをコピー
COPY --from=builder /app/main .
# 空のストレージディレクトリも持っていく
COPY --from=builder /app/storage /storage

# 8080ポートを使用することを明示
EXPOSE 8080

# 実行
CMD ["./main"]
