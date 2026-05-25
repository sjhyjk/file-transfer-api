# 変数定義（パスの変更に強くなります）
API_SPEC = api/openapi.yaml
API_GEN  = internal/handler/api.gen.go
API_CONFIG = api/config.yaml
API_PROTO = api/proto/file.proto

.PHONY: help
help: ## ヘルプを表示
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

.PHONY: gen-api
gen-api: ## OpenAPIからGoのコードを自動生成
	@echo "🚀 Generating Go code from $(API_SPEC)..."
	go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen -config $(API_CONFIG) $(API_SPEC)
	@echo "✨ Generation completed: $(API_GEN)"

.PHONY: gen-proto
gen-proto: gen-proto-go gen-proto-python ## Go と Python の両方の proto コードを生成

.PHONY: gen-proto-go
gen-proto-go: ## Go 用の gRPC コードのみを生成
	@echo "🚀 Generating gRPC code from $(API_PROTO)..."
	# Go 用
	protoc \
		--proto_path=$(dir $(API_PROTO)) \
		--go_out=. --go_opt=module=file-transfer-api \
		--go-grpc_out=. --go-grpc_opt=module=file-transfer-api \
		$(notdir $(API_PROTO))
	@echo "✨ Generation completed"

.PHONY: gen-proto-python
gen-proto-python: ## Python 用の gRPC コードを生成
	@echo "🐍 Generating gRPC code for Python..."
	# 💡 コマンド実行前にディレクトリを作成し、__init__.pyを配置（これでエラーを防ぎます）
	mkdir -p ./python_rag_worker/app/pb
	touch ./python_rag_worker/app/pb/__init__.py
	cp $(API_PROTO) ./python_rag_worker/app/pb/

	# Python 用 (grpcio-tools が必要)
	python3 -m grpc_tools.protoc \
		--proto_path=./python_rag_worker \
		--python_out=./python_rag_worker \
		--grpc_python_out=./python_rag_worker \
		./python_rag_worker/app/pb/$(notdir $(API_PROTO))

	rm ./python_rag_worker/app/pb/$(notdir $(API_PROTO))
	@echo "✨ Generation completed"

# --- [将来のための備え] ---
# 開発スピードを上げたい、またはインターネット環境なしで生成したい場合は以下を使用
.PHONY: install-tools
install-tools: ## ツールをPCに直接インストール (実行には $GOPATH/bin へのパス通しが必要)
	@echo "🛠️ Installing oapi-codegen..."
	go install github.com/deepmap/oapi-codegen/v2/cmd/oapi-codegen@latest

.PHONY: gen-api-local
gen-api-local: ## インストール済みのツールを使って高速に生成
	@echo "⚡ Generating Go code using local binary..."
	oapi-codegen -config $(API_CONFIG) $(API_SPEC)

.PHONY: tidy
tidy: ## go.mod の整理
	go mod tidy

.PHONY: build
build: ## アプリケーションのビルド
	go build -o bin/api cmd/api/main.go

.PHONY: run
run: ## ローカルでの実行
	go run cmd/api/main.go
