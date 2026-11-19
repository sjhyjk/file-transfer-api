# ===========================================================================
# 🛠️ File Transfer API & Python RAG Worker 統合管理 Makefile
# ===========================================================================

# --- ⚙️ 変数定義 (パスの変更に強くなります) ---
API_SPEC = api/openapi.yaml
API_GEN  = internal/handler/api.gen.go
API_CONFIG = api/config.yaml
API_PROTO = api/proto/file.proto

# 💡 共通のエラーハンドリング関数（マクロ）を定義してメッセージを一本化
define check_installed
	@if [ -z "$(1)" ]; then \
		echo "❌ Error: $(2) is not installed in your local environment."; \
		echo "👉 Please run 'make install-lint-tools' to install it, or use 'make lint-docker'."; \
		exit 1; \
	fi
endef

# 💡 各コマンドがローカル環境に存在するかチェック
GOLANGCI_LINT_CMD := $(shell command -v golangci-lint 2>/dev/null)
GO_ARCH_LINT_CMD  := $(shell command -v go-arch-lint 2>/dev/null)

# 💡 ローカルの仮想環境（venv）内にツールがあればそれを優先し、なければシステム標準（CI等）を使う動的定義
RUFF := $(shell if [ -f ./python_rag_worker/venv/bin/ruff ]; then echo "./venv/bin/ruff"; else echo "ruff"; fi)
MYPY := $(shell if [ -f ./python_rag_worker/venv/bin/mypy ]; then echo "./venv/bin/mypy"; else echo "mypy"; fi)
PYTEST := $(shell if [ -f ./python_rag_worker/venv/bin/pytest ]; then echo "./venv/bin/pytest"; else echo "pytest"; fi)

# 💡 Python用コマンド実行の共通プレフィックス（cdの重複を解消）
PY_RUN := cd python_rag_worker &&

.PHONY: help
help: ## ヘルプを表示
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# ---------------------------------------------------------------------------
# 🛠️ 1. Tools & Environment Setup (ツール & 仮想環境セットアップ)
# ---------------------------------------------------------------------------
.PHONY: python-env install-lint-tools install-tools install-grpcurl

python-env: ## Python の仮想環境 (venv) を自動構築し、依存ツールをインストール
	@if [ ! -d "python_rag_worker/venv" ]; then \
		echo "🐍 Creating Python virtual environment..."; \
		python3 -m venv python_rag_worker/venv; \
		python_rag_worker/venv/bin/python3 -m pip install --upgrade pip; \
		python_rag_worker/venv/bin/python3 -m pip install -r python_rag_worker/requirements.txt grpcio-tools mypy-protobuf pytest; \
	fi

install-lint-tools: ## ローカル解析ツールをPCに直接一括インストール
	@echo "🛠️ Installing golangci-lint..."
	mkdir -p /tmp/gcl_install
	cd /tmp/gcl_install && \
	curl -sSLo golangci-lint.tar.gz \
	https://github.com/golangci/golangci-lint/releases/download/v2.12.2/golangci-lint-2.12.2-linux-amd64.tar.gz && \
	tar xzf golangci-lint.tar.gz && \
	cp golangci-lint-2.12.2-linux-amd64/golangci-lint $$(go env GOPATH)/bin/ && \
	rm -rf /tmp/gcl_install
	@echo "🛠️ Installing go-arch-lint..."
	go install github.com/fe3dback/go-arch-lint@latest
	@echo "🛠️ Installing Python tools into venv..."
	$(PY_RUN) venv/bin/python3 -m pip install -r ../internal/tools/analysis/requirements-dev.txt

install-tools: ## ツールをPCに直接インストール (実行には $GOPATH/bin へのパス通しが必要)
	@echo "🛠️ Installing oapi-codegen..."
	go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest

install-grpcurl: ## gRPCのデバッグ用CLIツール (grpcurl) をインストール
	@echo "🛠️ Installing grpcurl..."
	go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest
	@echo "✨ grpcurl installed successfully. (Make sure $GOPATH/bin is in your PATH)"

# ---------------------------------------------------------------------------
# 🚀 2. Code Generation (自動生成タスク)
# ---------------------------------------------------------------------------
.PHONY: gen-api gen-proto gen-proto-go gen-proto-python gen-api-local

gen-api: ## OpenAPIからGoのコードを自動生成
	@echo "🚀 Generating Go code from $(API_SPEC)..."
	go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen -config $(API_CONFIG) $(API_SPEC)
	@echo "✨ Generation completed: $(API_GEN)"

gen-proto: gen-proto-go gen-proto-python ## Go と Python の両方の proto コードを生成

gen-proto-go: ## Go 用の gRPC コードを生成
	@echo "🚀 Generating gRPC code from $(API_PROTO)..."
	# Go 用
	protoc \
		--proto_path=$(dir $(API_PROTO)) \
		--go_out=. --go_opt=module=file-transfer-api \
		--go-grpc_out=. --go-grpc_opt=module=file-transfer-api \
		$(notdir $(API_PROTO))
	@echo "✨ Generation completed"

gen-proto-python: ## Python 用の gRPC コードを生成
	@echo "🐍 Generating gRPC code for Python from $(API_PROTO)..."
	# 🚀 正の proto ファイルを、Python 側の生成用ターゲットとして一時的にコピー配置
	cp $(API_PROTO) ./python_rag_worker/app/pb/

	# Python 用 (grpcio-tools が必要)
	$(PY_RUN) PATH="$$(pwd)/venv/bin:$$PATH" \
	./venv/bin/python3 -m grpc_tools.protoc \
		--proto_path=app \
		--python_out=app \
		--grpc_python_out=app \
		--mypy_out=app \
		--mypy_grpc_out=app \
		app/pb/$(notdir $(API_PROTO))
	
	# 🔥 生成が終わったら、二重管理にならないよう元の .proto コピーだけを綺麗に削除
	rm ./python_rag_worker/app/pb/$(notdir $(API_PROTO))
	@echo "✨ Generation completed"

gen-api-local: ## インストール済みのツールを使って高速に生成
	@echo "⚡ Generating Go code using local binary..."
	oapi-codegen -config $(API_CONFIG) $(API_SPEC)

# ---------------------------------------------------------------------------
# 🔍 3. Static Analysis (静的解析)
# ---------------------------------------------------------------------------
.PHONY: lint lint-go lint-python format-python lint-docker

lint: lint-go lint-python ## 【推奨】ローカル環境のネイティブツール(Go/venv)で全解析を実行
	$(call check_installed,$(GO_ARCH_LINT_CMD),go-arch-lint)
	@echo "📐 Running go-arch-lint locally..."
	@go-arch-lint check
	@echo "✨ All local linting checks finished successfully."

lint-docker: ## 【コンテナ検証用】Docker (static-analysis) を使って全解析を実行
	@echo "🐳 Running golangci-lint via Docker (Fallback Mode)..."
	docker run --rm -v $(CURDIR):/app -w /app golangci/golangci-lint:latest-alpine golangci-lint run ./...
	@echo "📐 & 🐍 Running Custom Linters (go-arch-lint & Python Quality Check) via Docker..."
	docker compose run --rm static-analysis

# --- 🐹 Go静的解析 ---
.PHONY: lint-go

lint-go: ## Goのコード品質チェック（golangci-lint）を実行
	@echo "🐹 Checking Go code with golangci-lint..."
	$(call check_installed,$(GOLANGCI_LINT_CMD),golangci-lint)
	@golangci-lint run ./...

# --- 🐍 Python静的解析 & 整形 ---
.PHONY: lint-python format-python

lint-python: ## Python用の静的解析と型チェックを実行
	@echo "🐍 Running Ruff Linter..."
	@$(PY_RUN) $(RUFF) check app
	@echo "🐍 Running Ruff Formatter (Check mode)..."
	@$(PY_RUN) $(RUFF) format --check app
	@echo "🐍 Running mypy Type Checker..."
	@$(PY_RUN) $(MYPY) app

format-python: ## Python用のコード自動整形を実行
	@echo "🧹 Formatting Python code with Ruff..."
	$(PY_RUN) $(RUFF) format app
	$(PY_RUN) $(RUFF) check app --fix

# ---------------------------------------------------------------------------
# 🛡️ 4. Security Scans (セキュリティスキャン)
# ---------------------------------------------------------------------------
.PHONY: trivy

trivy: ## Trivyによるコンテナイメージ/ファイルシステムの脆弱性スキャンを実行
	@echo "🛡️ Scanning vulnerabilities with Trivy..."

	@echo "🔍 [1/3] Scanning Go API Image..."
	@docker run --rm -v /var/run/docker.sock:/var/run/docker.sock -v $(CURDIR)/.cache:/root/.cache/ aquasec/trivy:latest image file-transfer-api-api:latest || true
	
	@echo "🔍 [2/3] Scanning Python RAG Worker Image..."
	@docker run --rm -v /var/run/docker.sock:/var/run/docker.sock -v $(CURDIR)/.cache:/root/.cache/ aquasec/trivy:latest image file-transfer-api-rag-worker-http:latest || true
	
	@echo "🔍 [3/3] Scanning Local File System (Vulnerabilities & Hardcoded Secrets)..."
	@docker run --rm -v $(CURDIR):/apps -v $(CURDIR)/.cache:/root/.cache/ aquasec/trivy:latest filesystem --scanners vuln,secret /apps || true

# ---------------------------------------------------------------------------
# 🧪 5. Test & Pipeline Triggers (自動テスト・クイック疎通確認)
# ---------------------------------------------------------------------------
.PHONY: test test-go test-python test-pubsub-publish test-http-trigger

test: test-go test-python ## Go と Python の両方の全テストをローカルで実行

test-go: tidy ## Goの単体テストを実行
	@echo "🐹 Running Go tests..."
	go test -v ./...

test-python: ## Pythonの単体テスト（pytest）を実行
	@echo "🐍 Running Python tests..."
	$(PY_RUN) $(PYTEST)

test-pubsub-publish: ## 【デバッグ用】Pub/Subエミュレータにテスト用のメッセージを1発直接パブリッシュする
	@echo "🚀 Publishing test message to Pub/Sub Emulator..."
	curl -X POST http://localhost:8085/v1/projects/local-project/topics/file-ingest-topic:publish \
		-H "Content-Type: application/json" \
		-d '{"messages": [{"data": "eyJmaWxlX25hbWUiOiAidGVzdC50eHQiLCAidGVuYW50X2lkIjogInRlbmFudC0wMDEifQ=="}]}'

test-http-trigger: ## 【デバッグ用】ローカルのFastAPI(HTTP Mode)にテストリクエストを送信
	curl -X POST http://localhost:8081/ingest \
		-H "Content-Type: application/json" \
		-d '{"file_name": "test.txt", "tenant_id": "tenant-001"}'

# ---------------------------------------------------------------------------
# 💻 6. Local Native Development (ホスト上のネイティブプロセス実行)
# ---------------------------------------------------------------------------

# --- 🚀 ハイブリッド一括起動 (Go ＋ Python 同時) ---
.PHONY: run-local-all-grpc-clean run-local-all-http-clean \
		run-local-all-grpc run-local-all-http

run-local-all-grpc-clean: kill-ports run-local-all-grpc ## ポートを解放してから gRPC 連携モードで一括起動

run-local-all-http-clean: kill-ports run-local-all-http ## ポートを解放してから HTTP Webhook 連携モードで一括起動

run-local-all-grpc: tidy ## 【ハイブリッド】Go(HTTP) と Python(gRPC) のローカルプロセスを両方一括で同時起動
	@echo "🚀 Starting both Go and Python(gRPC) locally..."
	@make run-python-grpc > /dev/null 2>&1 & echo $$! > .python.pid
	@sleep 1 # Pythonの起動をわずかに待つ
	@make run-local-grpc
	@make stop-local

run-local-all-http: tidy ## 【ハイブリッド】Go(HTTP) と Python(HTTP) のローカルプロセスを両方一括で同時起動
	@echo "🚀 Starting both Go and Python(HTTP) locally..."
	@make run-python-http > /dev/null 2>&1 & echo $$! > .python.pid
	@sleep 1
	@make run-local-http
	@make stop-local

# --- 🐹 Go単体ローカル起動 ---
.PHONY: run-local-grpc run-local-http run-local-pubsub

run-local-grpc: tidy ## 【Go単体】Go ➡️ Python(gRPC) パイプラインで起動
	@echo "💻 Starting Go Server in Local gRPC Pipeline Mode..."
	STORAGE_TYPE=LOCAL DB_TYPE=INMEMORY PIPELINE_TYPE=GRPC SERVER_MODE=HTTP go run cmd/api/main.go

run-local-http: tidy ## 【Go単体】Go ➡️ Python(HTTP) パイプラインで起動
	@echo "💻 Starting Go Server in Local HTTP Mode..."
	STORAGE_TYPE=LOCAL DB_TYPE=INMEMORY PIPELINE_TYPE=HTTP SERVER_MODE=HTTP go run cmd/api/main.go

run-local-pubsub: tidy ## 【Go単体】Go ➡️ Pub/Subエミュレータ連携モードで起動
	@echo "💻 Starting Go Server in Local GCP/PubSub Pipeline Mode..."
	@if [ -z "$$PUBSUB_EMULATOR_HOST" ]; then \
		echo "⚠️ PUBSUB_EMULATOR_HOST is not set. Defaulting to localhost:8085"; \
		export PUBSUB_EMULATOR_HOST="localhost:8085"; \
	fi; \
	STORAGE_TYPE=LOCAL DB_TYPE=INMEMORY PIPELINE_TYPE=GCP HTTP_PORT=8080 GCP_PROJECT_ID=local-project PUBSUB_TOPIC_ID=file-ingest-topic go run cmd/api/main.go

# --- 🐍 Python単体ローカル起動 ---
.PHONY: run-python-grpc run-python-http run-python-pubsub

run-python-grpc: ## 【Python単体】gRPCモードで直接起動
	@echo "🐍 Starting Python RAG Worker in [gRPC Mode]..."
	$(PY_RUN) GRPC_PORT=50051 STORAGE_ROOT=./storage python3 app/main_grpc.py

run-python-http: ## 【Python単体】HTTPモード(Uvicorn)で直接起動
	@echo "🐍 Starting Python RAG Worker in [HTTP Mode] (Uvicorn)..."
	$(PY_RUN) HTTP_PORT=8081 STORAGE_ROOT=./storage uvicorn main_http:app --host 0.0.0.0 --port 8081

run-python-pubsub: ## 【Python単体】Pub/Subメッセージ駆動モードで直接起動
	@echo "🐳 Ensuring Pub/Sub Emulator is running..."
	@docker compose up -d pubsub-emulator pubsub-init
	@echo "⏳ Waiting a moment for Emulator API to be stable..."
	@sleep 3
	@echo "🐍 Starting Python RAG Worker in [Pub/Sub Mode]..."
	@if [ -z "$$PUBSUB_EMULATOR_HOST" ]; then \
		echo "⚠️ Warning: PUBSUB_EMULATOR_HOST is not set. Defaulting to 127.0.0.1:8085"; \
		export PUBSUB_EMULATOR_HOST="127.0.0.1:8085"; \
	else \
		if [ "$$PUBSUB_EMULATOR_HOST" = "localhost:8085" ]; then \
			export PUBSUB_EMULATOR_HOST="127.0.0.1:8085"; \
		fi; \
	fi; \
	$(PY_RUN) GCP_PROJECT_ID=local-project PUBSUB_SUBSCRIPTION_ID=file-ingest-sub STORAGE_ROOT=./storage python3 app/main_pubsub.py

# --- 🧹 ネイティブプロセス終了・ポート管理 ---
.PHONY: stop-local kill-ports

stop-local: ## バックグラウンド起動した Python などのローカルプロセスを安全に停止
	@echo "🛑 Stopping background local processes..."
	@if [ -f .python.pid ]; then \
		kill $$(cat .python.pid) 2>/dev/null || true; \
		rm -f .python.pid; \
	fi
	@echo "✨ Local processes stopped."

kill-ports: ## 開発用ポート(8080, 50050)を強制解放
	@echo "🛑 Killing processes on dev ports..."
	@fuser -k 8080/tcp || true
	@fuser -k 50050/tcp || true
	@echo "✨ Ports 8080 and 50050 have been cleared."

# ---------------------------------------------------------------------------
# 🐳 7. Docker Environment Control (Docker Compose 統合制御)
# ---------------------------------------------------------------------------

# --- 🚀 全体起動・構築系 ---
.PHONY: dev up up-fg up-build restart setup-emulator-resources

dev: ## 【最強コンボ】完全クリーン起動 ＋ 自動リソース注入 ＋ ログ追跡
	# 💡 起動前に必ず storage ディレクトリの存在と権限を担保する
	@mkdir -p storage
	@chmod 777 storage || true
	
	# 既存のコンテナ停止 & 起動処理
	@make down
	@export DOCKER_BUILDKIT=1; export COMPOSE_DOCKER_CLI_BUILD=1; make up-build
	@make setup-emulator-resources || true
	@echo "📡 All components and emulator resources are ready!"
	@make logs

# --- 🐳 Docker Compose (マルチコンテナ一括制御) ---

up: ## Dockerコンテナをバックグラウンドで一括起動 (DB, Go, Python×3, Emulator)
	@echo "🐳 Starting all containers in background..."
	docker compose up -d
	@echo "✨ Containers are up. Use 'make logs' to watch."

up-fg: ## Dockerコンテナを一括起動 (フォアグラウンド / ログを直接確認)
	docker compose up

up-build: ## 【推奨】キャッシュを無視してビルドし、バックグラウンドで起動
	@echo "🐳 Building and starting all containers..."
	@COMPOSE_DOCKER_CLI_BUILD=1 DOCKER_BUILDKIT=1 docker compose up --build -d

restart: down up ## コンテナ環境を完全に再起動

setup-emulator-resources: ## 起動中のPub/Subエミュレータにトピックとサブスクリプションを注入
	@echo "☁️ Creating Topic & Subscription inside Pub/Sub Emulator..."
	curl -X PUT http://localhost:8085/v1/projects/local-project/topics/file-ingest-topic \
		-H "Content-Type: application/json"
	curl -X PUT http://localhost:8085/v1/projects/local-project/subscriptions/file-ingest-sub \
		-H "Content-Type: application/json" \
		-d '{"topic": "projects/local-project/topics/file-ingest-topic"}'
	@echo "\n✨ Pub/Sub Local Resources are successfully provisioned."

# --- 🎯 特定サービス個別制御 (引数 `svc` を指定して利用) ---
# 使い方例: make restart-svc svc=api
.PHONY: build-svc up-svc stop-svc restart-svc

build-svc: ## 【単体】特定コンテナをキャッシュ無視で強制再ビルド (例: make build-svc svc=api)
	@if [ -z "$(svc)" ]; then echo "❌ Error: 'svc' argument is required. (e.g., make build-svc svc=api)"; exit 1; fi
	@echo "🔧 Rebuilding container [$(svc)]..."
	docker compose build --no-cache $(svc)

up-svc: ## 【単体】特定コンテナをバックグラウンド起動 (例: make up-svc svc=api)
	@if [ -z "$(svc)" ]; then echo "❌ Error: 'svc' argument is required."; exit 1; fi
	@echo "🚀 Starting container [$(svc)]..."
	docker compose up -d $(svc)

stop-svc: ## 【単体】特定のコンテナだけを安全に停止 (例: make stop-svc svc=api)
	@if [ -z "$(svc)" ]; then echo "❌ Error: 'svc' argument is required."; exit 1; fi
	@echo "🛑 Stopping container [$(svc)]..."
	docker compose stop $(svc)

restart-svc: ## 【単体最速リビルド】停止 ➔ ビルド ➔ 再起動 ➔ ログ追跡を一気通貫 (例: make restart-svc svc=api)
	@if [ -z "$(svc)" ]; then echo "❌ Error: 'svc' argument is required."; exit 1; fi
	@make stop-svc svc=$(svc)
	@make build-svc svc=$(svc)
	@make up-svc svc=$(svc)
	@echo "✨ Container [$(svc)] successfully restarted."
	@make logs-svc svc=$(svc)

# --- 📜 ログ確認系 ---
.PHONY: logs logs-svc

logs: ## 全コンテナのログをリアルタイムで追跡 (Ctrl+C で終了)
	docker compose logs -f

logs-svc: ## 【単体】特定コンテナのログをリアルタイム追跡 (例: make logs-svc svc=api)
	@if [ -z "$(svc)" ]; then echo "❌ Error: 'svc' argument is required."; exit 1; fi
	docker compose logs -f $(svc)

# --- 🛑 停止・クリーンアップ系 ---
.PHONY: down

down: ## 全てのコンテナを停止・削除し、ネットワークもクリーンアップ
	@echo "🐳 Stopping and removing all containers..."
	docker compose down

# ---------------------------------------------------------------------------
# 🗄️ 8. Database Controls (データベース個別制御・初期化)
# ---------------------------------------------------------------------------
.PHONY: db-psql db-reset

db-psql: ## 起動中の PostgreSQL コンテナに psql で直接インタラクティブに入る
	docker compose exec db psql -U postgres -d postgres

db-reset: ## 【開発用】DBコンテナのボリュームを削除し、データとスキーマを完全クリーンにして再起動
	@echo "🚨 Resetting Database storage and volume..."
	docker compose down db -v
	docker compose up -d db
	@echo "⏳ Waiting for DB to be ready..."
	@sleep 3
	@echo "🔄 Re-running migrations via Go API container..."
	docker compose up -d api

# ---------------------------------------------------------------------------
# 🐹 9. Go Utilities & Lifecycle (依存管理・クリーンアップ)
# ---------------------------------------------------------------------------
.PHONY: tidy clean clean-docker

tidy: ## go.mod と go.sum の整合性をチェックして整理
	go mod tidy

clean: ## ホスト側が生成したMypy/Ruff等の静的解析キャッシュを安全に全削除
	@echo "🧹 Cleaning up caches..."
	rm -rf .mypy_cache .ruff_cache
	rm -rf python_rag_worker/.mypy_cache python_rag_worker/.ruff_cache
	@echo "✨ Clean completed."

clean-docker: ## 【安全】Dockerが生成したキャッシュをsudoなしで安全にコンテナ経由で全削除
	@echo "🧹 Cleaning up caches securely via lightweight container..."
	docker run --rm -v $(CURDIR):/app -w /app alpine rm -rf .mypy_cache .ruff_cache \
		python_rag_worker/.mypy_cache python_rag_worker/.ruff_cache
	@echo "✨ Clean completed."
