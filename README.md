# Go Parallel File Transfer API (Architecture Study)

![Build Status](https://github.com/sjhyjk/file-transfer-api/actions/workflows/docker-build.yml/badge.svg)

## 📝 プロジェクトの背景と目的
<div align="center">
  <h3><b>「インフラへのアプリケーションの埋没」を打破し、ビジネスロジックを解き放つ</b></h3>
</div>

---

本プロジェクトの原点は、現職の金融基盤で直面した「**インフラ制約によるビジネス要件の断念**」という強い課題意識にあります。

#### 🚩 直面した 3 つの構造的課題
* **プラットフォーム依存による技術選定の硬直化**: 基盤側の都合でモダンな進化的アーキテクチャが阻害される現状。

* **共有リソース基盤におけるオーケストレーションの限界**: 共通基盤ゆえにテナント個別の要件 (断続処理など) を許容できない柔軟性の欠如。

* **脆弱なセキュリティ境界**: 共有ストレージにおける「他テナントのパス推測可能性」という構造的欠陥。

これらを「反面教師」とし、**エンタープライズ基準の厳格な制約下**でも進化可能なポータビリティと、開発者がロジックに集中できる**開発者体験(DevEx)**の両立を、アーキテクチャの力で証明することを目的としています。

## 🚀 実装の柱

### 🏛️ スキーマ駆動開発 & クリーンアーキテクチャによる疎結合設計
OpenAPI 3.0 および **Protocol Buffers** を **SSOT** とし、コード生成（oapi-codegen / protoc）による型安全な実装を強制。DIP の徹底によりビジネスロジックをインフラから隔離し、 `go-arch-lint` を用いた **Architecture Testing** により、設計の腐敗を静的に遮断しています。

- **Transport Layer (Echo & gRPC)**: `Echo` および `gRPC` を併用し、共通の Interactor を介したマルチプロトコル・アダプター構成を採用。gRPC では **Reflection API** を有効化し、動的なサービス探索に対応。Middleware/Interceptor による **OIDC 認証** と **Trace ID 注入** を統合し、入り口でのガバナンスを共通化。
- **Domain**: 唯一の **Source of Truth**。数学的な「定義の抽象化」を意識し、インフラの制約から独立した**不変条件(Invariants)**をインターフェースとして定義。全レイヤーの依存が向かう「不動の頂点」として配置しています。
- **Usecase**: `errgroup` を活用した **Fail-fast な並行処理制御** を実装。`infra` 層の具象実装を一切参照せず、純粋なビジネスロジック（並行アップロードのオーケストレーション等）に特化。
- **Infrastructure (Factory Pattern)**: 環境変数に基づき、GCS/Local Storage/S3（予定）、Cloud SQL/In-Memory DB を動的に切り替える **Plug-and-Play** な構成を採用。
- **cmd (Main Component)**: 依存注入（DI）と起動に特化。API/gRPC/CLI といった実行形態を外部から注入可能にし、コアロジックの再利用性を最大化。
- **Observability**: `slog` を拡張し、**REST / gRPC 両プロトコルを横断**して Trace ID を `context` 経由で伝播。並行処理下でも、リクエスト単位のログ追跡を完全に実現。
- **Ingest Pipeline (External Integration)**: ファイルのライフサイクル（保存完了）をトリガーとした外部通知基盤を構築。DIPに基づき `domain` 層で定義されたインターフェースを介して、Python RAG基盤等の外部システムと疎結合に連携します。

## ⚡ Go の並行処理モデルの実測検証

同一コンテナ環境（リソース制限下）において、逐次処理と並行処理のスループットを実測比較しました。

### 検証条件
- **比較対象**: `UploadMultipleSerial` (逐次) vs `UploadMultipleParallel` (Goroutine)
- **環境**: 同一の Docker コンテナ内から GCS への転送
- **データ**: 3つのファイル同時転送

### 📈 Performance Benchmark Results (Latest)
| 方式 | 実行時間 | 備考 |
|:--- |:--- |:--- |
| **Method A (Serial)** | **1.686s** | 逐次アップロード |
| **Method B (Parallel)** | **0.627s** | **Goroutine による並行最適化** |

**→ パフォーマンス改善率: 62.77%**

#### 📊 プロフェッショナル・ベンチマーク (Go standard `testing.B` による実測)
モック環境下での10ファイル同時処理コスト（530回の試行平均）:
- **Average Latency**: **2.56 ms / op**
- **Memory Efficiency**: **1,427 B / op** (極めて低メモリな実行を実現)
- **Allocations**: **43 allocs / op**

### 📝 設計判断への活用
本検証により、Go の軽量スレッド（Goroutine）を活用することで、**インフラ構成を変更せずに I/O ボトルネックを構造的に解消可能**であることを実証しました。この定量的なデータは、将来的な大規模データインジェスト基盤の設計における重要な判断材料となります。

## 🛠 検証済み環境 (Verified Infrastructure)

本システムは、以下のマネージドサービス構成において正常動作およびパフォーマンス計測を完了しています。

| Category | Specification |
|:--- |:--- |
| **Compute** | **Cloud Run** (First Generation / 512MiB Memory / 1 vCPU) |
| **Storage** | **Google Cloud Storage** (Standard Tier / Region: us-west1) |
| **Database** | **Cloud SQL for PostgreSQL** (v15 / Shared CPU / 10GB Storage) |
| **Networking** | **Unix Domain Socket** via Cloud SQL Auth Proxy (Private Connectivity) |
| **Security** | **Workload Identity / ADC** (Service Account Keyless Auth) |

- **CI/CD**: GitHub Actions による完全自動化（Artifact Registry 連携）
- **DB Migration**: `golang-migrate` による起動時オートマイグレーション

## 🏗 検証済みロードマップ (Infrastructure & Backend)

📂 Schema-First & API Governance

- [x] **マルチプロトコル・スキーマ駆動開発 (SSOT)** 🎉 Done
  - **Single Source of Truth**: OpenAPI 3.0 および Protocol Buffers を採用。仕様と実装の乖離を構造的に排除。
  - **gRPC Reflection の導入**: サービス定義を動的に公開。`grpcurl` 等による高速なデバッグサイクルを確立。
  - **自動生成の統合**: `oapi-codegen` および `protoc` による型安全な Go コード生成をパイプライン化。

📂 Database & Persistence Strategy

- [x] **DB 永続化とトランザクション整合性の管理** 🎉 *Done*
  - **高度な検索実装**: **Specification Pattern** を導入。PostgreSQL の**配列演算子・GINインデックス**による動的かつ高速なフィルタリングを実現。プレースホルダによる動的クエリ構築により、SQLインジェクションを完全に排除。
  - **整合性担保**: **補償トランザクション**を実装し、DB保存失敗時の GCS ロールバックを自動化。`pgxpool` と Unix ドメインソケットを用いたセキュアな接続基盤を構築。
  - **クリーンなAPI設計**: HTTP クエリパラメータを Domain 層の抽象型へ変換する **「マルチプロトコル対応」** の玄関口を実装。`limit/offset` によるページネーションバリデーションを全レイヤーで統合。

- [x] **DBマイグレーションの自動化** 🎉 *Done*
  - **指数バックオフによる接続リトライ**: DBコンテナの起動遅延を許容するリトライ戦略を実装。
  - **Single Binary Strategy**: `golang-migrate` と `io/fs`(embed) を活用し、バイナリ内包型の自動マイグレーションを実現。環境差分による不具合を**仕組みで排除**。

⚙️ CI/CD & Cloud Native

- [x] **GitHub Actions による高度な CI 構築** 🎉 *Done*
  - **安全性と性能の自動化**: `go test` による自動テスト、`testing.B` による性能監視、および商用グレードのリンターによる**機密情報混入の静的検知**を統合。
  - **運用最適化**: `workflow_dispatch` を導入し、コストや状況に応じた**柔軟な手動デプロイ制御(If-conditional flow)** を確立。

- [x] **Cloud Run への自動デプロイ (CD)** 🎉 *Done*
  - **Attack Surface 最小化**: **Distroless** イメージを採用し、実行環境の脆弱性リスクを根本から低減。
  - **Credential Zero (OIDC)**: Artifact Registry 連携と **Workload Identity** による **Keyless 認証 (OIDC)** を確立。サービスアカウントキーの管理を完全に廃止し、認証情報のバイナリ内包を完全に排除。
  - **Container-native Base**: `docker-compose` による API+DB のローカル完結型開発環境を構築。

- [x] **IaC 化 (Terraform)による再現性の確保** 🎉 *Done*
  - **構成同期（Drift Detection）**: 既存リソースの状態をコードへ正確に反映。**Drift（環境差分）を完全に解消**し、コードと実環境の完全な同期を完遂。
  - **最小権限の原則 (Least Privilege)**: サービスアカウントに対し、実行時（Object Admin）と管理時（Storage Admin）の権限を分離。セキュアな IAM 設計を実証。
  - **ライフサイクル管理**: `force_destroy` 等の属性定義により、リソースの廃棄・再作成プロセスを宣言的に記述。

🏗 Architecture & Reliability

- [x] **Architecture Testing (理論駆動の実証)** 🎉 *Done*
  - **静的解析の強制**: `go-arch-lint` により **DIP (依存性逆転の原則)** を静的に強制。設計の腐敗を自動で遮断する仕組みを構築。
  - **品質管理の自動化**: `golangci-lint` を導入。`errcheck` や `staticcheck` 等により商用レベルのコード品質を担保。

- [x] **オブザーバビリティ & 並行処理制御 (Distributed Tracing)** 🎉 *Done*
  - **Trace ID 伝播**: HTTP Middleware で生成した Trace ID を `context` 経由で伝播。`slog.Handler` を拡張し、全ログ行への `trace_id` 自動刻印を完遂。
  - **並行処理の Fail-fast**: `errgroup` を用いた異常検知時の即座な処理中断を実装。

- [x] **マルチプロトコル・アダプターの構築** 🎉 *Done*
  - **Handler Layer の分離**: 同一の Interactor (Usecase) を Echo (REST) と gRPC の両ハンドラで共用。プロトコル非依存の純粋なビジネスロジックを隔離。
  - **共通基盤の統合**: HTTP Middleware と gRPC Interceptor で共通の Trace ID 伝播・ログ・エラーハンドリングを実現。

- [x] **RAG / データインジェスト基盤への統合（通知パイプライン）** 🎉 *Done*
  - **イベント駆動型通知**: ファイル保存完了をトリガーに、外部（Python RAG基盤等）へメタデータを即時通知する `http_notifier` を実装。
  - **ライフサイクル管理**: `FileMetadata` に `Status` (Pending/Processing/Completed/Failed) を導入。非同期なデータ処理状態の追跡を可能に。
  - **プロバナンス（由来）情報の保持**: `Source` フィールドを追加し、RAG における回答精度の向上に不可欠な「情報のソース追跡」をスキーマレベルでサポート。

- [x] **Polyglot Communication (Go ↔ Python 連携)** 🎉 Done
  - **マルチ言語構成**: Go（API）と Python（RAG Mock）を Docker Compose ネットワーク内で統合。`http_notifier` による言語間通信の実証を完遂。
  - **コンテナオーケストレーション**: `depends_on` による起動順序制御と、ネットワークエイリアスを用いたサービス間名前解決を確立。

## 🛠 今後の検証ロードマップ

### Phase 1: RAG 拡張 & イベント駆動コア（最優先）

- [x] **ドメイン駆動 RAG パイプラインの実装** 🎉 *Done*
  - ファイル保存をトリガーとした「テキスト抽出 → チャンク分割 → ベクトル化」のフローをドメインイベントとして定義。
  - `python_rag_worker` を真の処理基盤へ昇格させ、非同期的なデータ加工パイプラインを確立。

- [ ] **イベント駆動アーキテクチャの導入 (Pub/Sub 抽象化)**
  - Pub/Sub を基盤としたメッセージングモデルの定義。

### Phase 2: エンタープライズ・リライアビリティ

- [ ] **Redis による分散ロックと冪等性制御**
  - 重いバッチ処理や外部通知における二重実行を防止する「Idempotency Control」の実装。

- [ ] **テナントごとの流量制御 (Rate Limiting)**
  - 特定テナントの負荷がシステム全体を阻害しないよう、Redis を用いたクォータ管理を導入。

### Phase 3: プラットフォーム・ガバナンス & DevEx

- [ ] **マルチテナント隔離の完全防衛**
  - 物理ストレージパスの隠蔽とアクセス認可（IDOR対策）とメタデータ紐付けによる、他テナントからの「パス推測」を完全に遮断するストレージアダプターの実装。

- [ ] **OpenTelemetry による分散トレーシングの高度化**
  - Go と Python を跨ぐリクエストの可視化、およびテナントごとのリソース消費メトリクスの露出。

- [ ] **管理用ダッシュボード (React) & CLI の提供**
  - 処理ステータスの可視化、および開発者向けの自動生成（Scaffolding）ツールの統合。

## 📁 プロジェクト構造
```text
.
├── api/                   # API 定義・スキーマ（情報のソースオブトゥルース）
│   ├── proto/             # gRPC 定義
│   │   └── file.proto     # gRPC/Protocol Buffers 用の定義
│   ├── config.yaml        # サービス設定・環境定義
│   └── openapi.yaml       # REST API (OpenAPI) 仕様書
├── cmd/                   # Entry Point (実行環境の決定・DI・起動)
│   ├── api/
│   │   └── main.go        # 本番サーバー起動（DI、gRPC/REST並行起動）
│   └── benchmark/
│       └── main.go        # 性能比較検証用（Serial vs Parallel 計測）
├── internal/              # Business Logic (クリーンアーキテクチャのコア)
│   ├── domain/            # Entity & Repository Interface (DIPの起点)
│   │   ├── file.go        # ファイルの実体（Entity）
│   │   ├── repository.go  # 保存(Repo)と通知(Pipeline)の定義
│   │   └── metadata.go    # RAG連携用の属性定義
│   ├── usecase/           # Business Logic (並行処理・制御フロー)
│   │   ├── file_interactor.go       # Goの並行アップロードのコアロジック
│   │   └── file_interactor_test.go  # ロジックの正当性を保証するテスト
│   ├── handler/           # 外部接続（HTTPリクエストの解析・レスポンス生成）
│   │   ├── api.gen.go     # OpenAPIから自動生成されたボイラープレート
│   │   ├── grpc/          # gRPC ハンドラー実装 & Reflection 制御
│   │   │   ├── server.go            # gRPC サーバー起動・管理
│   │   │   ├── file_handler_grpc.go # gRPC ハンドラー本体
│   │   │   └── pb/        # protoc生成ファイル
│   │   │       ├── file.pb.go
│   │   │       └── file_grpc.pb.go
│   │   └── rest/          # Echo (REST) ハンドラー
│   │       ├── file_handler_http.go
│   │       ├── router.go      # ルーティング・ミドルウェア設定
│   │       └── appmiddleware/ # アプリ固有のミドルウェア（Trace ID注入等）
│   │           └── trace.go   # Trace ID注入等の共通前処理
│   ├── infra/             # Infrastructure Adapters (技術的詳細の実装)
│   │   ├── factory.go     # インフラ切り替えの司令塔
│   │   ├── pipeline/      # External Integration (RAG基盤等へのイベント通知)
│   │   │   ├── grpc_notifier.go  # Python API への gRPC 通知実装
│   │   │   └── http_notifier.go  # Python API への HTTP 通知実装
│   │   ├── gcs/           # GCS 具象実装（Workload Identity 対応） (STORAGE_TYPE=GCS)
│   │   │   └── gcs_repository.go
│   │   ├── local/         # ローカルファイルシステム実装 (STORAGE_TYPE=LOCAL)
│   │   │   └── local_repository.go
│   │   ├── sql/           # Cloud SQL (PostgreSQL) 永続化・マイグレーション
│   │   │   ├── client.go                # DB接続プール（pgxpool）の管理とリトライロジック
│   │   │   ├── postgres_repository.go   # Repository 構造体の定義と、共通処理・Closeなどのインフラ共通定義
│   │   │   ├── metadata_ops.go          # FileMetadata（CRUD）に関するデータ操作ロジック
│   │   │   └── migrations.go            # golang-migrate 実行ロジック
│   │   └── inmemory/  # 高速な検証を可能にするインメモリDB実装
│   │        └── memory_repository.go
│   └── pkg/               # ユーティリティ・基盤共通パッケージ
│       ├── config/        # 設定ロード (config.go)
│       └── logger/        # 構造化ログ（slog）基盤。Trace IDの伝播を管理
│           ├── context.go
│           └── handler.go
│
├── python_rag_worker/     # Python RAG 基盤
│   ├── app/               # アプリケーションの構成とDIの起点
│   │   ├── main_http.py             # HTTPワーカー専用エントリーポイント（FastAPI）
│   │   ├── main_grpc.py             # gRPCワーカー専用エントリーポイント（Async gRPC）
│   │   ├── api/           # ハンドラー層（外部リクエストの解釈とルーティング）
│   │   │   ├── http_handler.py      # HTTPエンドポイント・ヘルスチェック
│   │   │   ├── grpc_handler.py      # gRPCサービサーロジック
│   │   │   └── grpc_server.py       # gRPCサーバーの純粋なライフサイクル管理
│   │   ├── pb/            # Python 用 gRPC 生成コード
│   │   │   ├── file_pb2.py
│   │   │   └── file_pb2_grpc.py
│   │   ├── services/      # ユースケース・ドメイン層（パイプラインの定義）
│   │   │   └── rag_service.py       # オーケストレーションとRAG核心ロジック（HTTP/gRPC共通）
│   │   ├── infra/         # インフラ層（外部ライブラリの実装詳細）
│   │   │   ├── extractors/          # 形式ごとの抽出器を格納
│   │   │   │   ├── base.py          # 共通インターフェース
│   │   │   │   ├── excel_extractor.py
│   │   │   │   ├── pdf_extractor.py 
│   │   │   │   ├── pptx_extractor.py
│   │   │   │   ├── text_extractor.py
│   │   │   │   └── word_extractor.py 
│   │   │   ├── extractor_factory.py # 形式判定と振り分け
│   │   │   └── chunker.py           # チャンク分割
│   │   └── core/
│   │       ├── config.py            # 環境変数・設定管理
│   │       └── logger.py            # ログの一元初期化
│   ├── Dockerfile         # Python 実行環境のコンテナ定義
│   └── requirements.txt   # Python 依存ライブラリ
│
├── migrations/            # DB スキーマ管理 (SQLファイル)
│   ├── 000001_create_file_metadata_table.up.sql
│   ├── 000001_create_file_metadata_table.down.sql
│   ├── 000002_add_gin_index_to_tags.up.sql
│   └── 000002_add_gin_index_to_tags.down.sql
├── terraform/             # Infrastructure as Code (GCPリソース定義)
│   ├── main.tf            # GCSリソース・Provider定義
│   ├── outputs.tf         # インフラ出力情報の定義
│   └── variables.tf       # プロジェクトID・バケット名の変数管理
├── .github/workflows/     # CI/CD パイプライン (GitHub Actions)
│   └── docker-build.yml   # 自動コンテナビルド定義
├── data/                  # テスト用データ（parallel-test-*.txt 等）
│
# 💡 【Goコード・ビルド関連の塊】
├── assets.go              # プロジェクト共通資産（SQL等）の embed 定義
├── go.mod                 # 依存関係管理
├── tools/                 # 開発ツールの依存関係管理
│   └── tools.go           # ビルドツールのバージョンを固定するための Go ファイル
│
# 💡 【環境・コンテナインフラ関連の塊】
├── docker-compose.yml     # DB・API, Python Mockの疎結合な依存関係とネットワークを定義
├── Dockerfile             # マルチステージビルドによる軽量実行イメージ定義
├── .env                   # 環境設定（Git管理対象外）
│
# 💡 【Linter・CI/CD・タスク自動化関連の塊】
├── .go-arch-lint.yml      # クリーンアーキテクチャの依存方向を強制する Lint 定義
├── .golangci.yml          # 静的解析ルール定義
├── Makefile               # 標準化された開発コマンド (Go/Python 両対応)
├── README.md              # 本ドキュメント
│
├── python_comparison/     # [In Progress] Go (Parallel) vs Python (AsyncIO) のスループット・メモリ効率比較検証
└── aws_infrastructure/    # [In Progress] マルチクラウド (S3) 展開用の設計検討
```

## 🌐 データフロー戦略
```text

🌐 User/Client
      │
      ▼ [HTTPS/JSON] (GET /files?tags=...)
┌──────────────────────────┐
│  🚀 Google Cloud Run     │
│  (Go API Container)      │
│                          │
│  ┌────────────────────┐  │
│  │ ⚙️ Factory Pattern │  │
│  │ 🔐 ADC Auth        │  │
│  └────┬───────────┬───┘  │
└───────┼───────────┼──────┘
        │           │
        ▼ [gRPC]    ▼ [SQL/Unix Socket]
┌───────────────┐  ┌────────────────────┐
│ 📦 GCS Bucket │  │ 🐘 Cloud SQL (PG)  │
│ (File Binary) │  │ (File Metadata)    │
└───────────────┘  └────────────────────┘
```

## 🏗 アーキテクチャと依存関係の制御
```text

【依存方向のフロー】
🌐 External (API/CLI) ──┐
                         ▼
   ┌──────────────────────────────────────────┐
   │  cmd/ (Main/DI Container)                │
   └──────────┬───────────────────────────────┘
              │ 1. Instantiate & Inject
              ▼
   ┌──────────────────────────────────────────┐
   │  internal/usecase (Business Logic)       │
   └──────────┬───────────────┬───────────────┘
              │               │ 
              │ (Interface)   │ (Interface)
              ▼               ▼
   ┌─────────────────┐ ┌──────────────────────┐
   │ internal/domain │ │ internal/infra       │
   │ (Entity/Models) │ │ (Adapters/GCS/SQL)   │
   └─────────────────┘ └──────────────────────┘
              ▲               │
              └───────────────┘
                2. Implements Domain Interfaces

```

## 🛠 Quick Start (Local Development)

外部インフラ（GCP）に依存せず、`Makefile` と`Docker Compose` を用いて「API + DB + 擬似ストレージ」の全環境を即座にローカルで再現可能です。

1. 依存ツールのセットアップ & コード生成
OpenAPI スキーマから型安全な Go コードを生成します。

```bash
# 依存ライブラリの同期
go mod tidy

# OpenAPI スキーマからハンドラ・型定義を自動生成
make gen-api
```

2. 環境の起動
開発スタイルに合わせて、以下のいずれかの方法で起動してください。

```bash
### 🐳 A. Docker Compose (Recommended)
# DBリトライ・オートマイグレーションを含めたフル環境を起動
docker compose up --build

### 💻 B. Go Run (Lightweight Mode)
# 外部インフラに依存せず、インメモリDBとローカルストレージで起動
STORAGE_TYPE=LOCAL DB_TYPE=INMEMORY go run cmd/api/main.go
```

3. 動作確認（ヘルスチェック）
```bash
# システムの稼働状況と起動時パフォーマンス（Gain）を確認
curl http://localhost:8080/health
```

```text
> [!TIP]
**DBリトライ戦略**: DBコンテナの起動遅延を許容する「指数バックオフによるリトライアルゴリズム」を実装済みです。`docker compose` 実行時、依存関係の順序を問わず、サービスは自動で正常な状態に収束します。
```
