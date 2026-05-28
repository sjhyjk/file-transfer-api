# 変数定義（パスの変更に強くなります）
API_SPEC = api/openapi.yaml
API_GEN  = internal/handler/api.gen.go
API_CONFIG = api/config.yaml
API_PROTO = api/proto/file.proto

# 💡 ローカルの仮想環境（venv）内にツールがあればそれを優先し、なければシステム標準（CI等）を使う動的定義
RUFF := $(shell if [ -f ./python_rag_worker/venv/bin/ruff ]; then echo "./python_rag_worker/venv/bin/ruff"; else echo "ruff"; fi)
MYPY := $(shell if [ -f ./python_rag_worker/venv/bin/mypy ]; then echo "./python_rag_worker/venv/bin/mypy"; else echo "mypy"; fi)

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
gen-proto-go: ## Go 用の gRPC コードを生成
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
	@echo "🐍 Generating gRPC code for Python from $(API_PROTO)..."
	# 🚀 正の proto ファイルを、Python 側の生成用ターゲットとして一時的にコピー配置
	cp $(API_PROTO) ./python_rag_worker/app/pb/

	# Python 用 (grpcio-tools が必要)
	cd python_rag_worker/app && python3 -m grpc_tools.protoc \
		--proto_path=. \
		--python_out=. \
		--grpc_python_out=. \
		./pb/$(notdir $(API_PROTO))
	
	# 🔥 生成が終わったら、二重管理にならないよう元の .proto コピーだけを綺麗に削除
	rm ./python_rag_worker/app/pb/$(notdir $(API_PROTO))
	@echo "✨ Generation completed"

.PHONY: lint-python
lint-python: ## Python用の静的解析と型チェックを実行
	@echo "🐍 Running Ruff Linter..."
	$(RUFF) check python_rag_worker/app
	@echo "🐍 Running Ruff Formatter (Check mode)..."
	$(RUFF) format --check python_rag_worker/app
	@echo "🐍 Running mypy Type Checker..."
	$(MYPY) python_rag_worker/app

.PHONY: format-python
format-python: ## Python用のコード自動整形を実行
	@echo "🧹 Formatting Python code with Ruff..."
	$(RUFF) format python_rag_worker/app
	$(RUFF) check --fix python_rag_worker/app

# 🐳 Dockerコンテナを使って解析したい場合はこちらを叩く
.PHONY: docker-lint-python
docker-lint-python: ## Dockerコンテナ経由でPythonの静的解析を実行
	docker compose run --rm static-analysis ruff check .
	docker compose run --rm static-analysis mypy .

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
