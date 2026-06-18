# ==========================================
# Build Stage (コンパイル専用ステージ)
# ==========================================

FROM golang:1.25-bookworm AS builder

WORKDIR /app

# 1. 依存関係を先にコピーしてキャッシュを効かせる
COPY go.mod go.sum ./
RUN go mod download

# 2. 全ソースコピー（migrations等も含む）
COPY . .

# 3. 静的リンクしたバイナリをビルド（CGOをオフにして軽量化）
# キャッシュマウントを追加
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o main ./cmd/api/main.go

# 4. 書き込み用ディレクトリの事前作成（Distroless対策）
RUN mkdir -p /app/storage

# ==========================================
# Runtime Stage (実行専用ステージ)
# ==========================================

# 実行用には最小限のイメージを使用
FROM gcr.io/distroless/static-debian12

WORKDIR /

# 💡 安全のため、Distrolessに組み込まれている最小特権ユーザーに変更
USER nonroot:nonroot

# ビルドしたバイナリをコピー
COPY --from=builder --chown=nonroot:nonroot /app/main .
# 空のストレージディレクトリも持っていく
COPY --from=builder --chown=nonroot:nonroot /app/storage /storage
COPY --from=builder --chown=nonroot:nonroot /app/migrations /migrations

# 8080, 50050ポートを使用することを明示
EXPOSE 8080
EXPOSE 50050

# 実行
CMD ["./main"]
