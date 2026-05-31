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
OpenAPI 3.0 および **Protocol Buffers** を **SSOT** とし、Go/Python両層のコードおよび型定義（.pyi）を Makefile で完全自動生成。DIP の徹底によりビジネスロジックをインフラから隔離し、 Go側の `go-arch-lint/golangci-lint`、Python側の `Ruff/Mypy` を用いた **Architecture Testing** により、設計の腐敗を構造的に遮断しています。

- **Transport Layer (Echo & gRPC)**: 同一の Interactor を共有するマルチプロトコル・アダプター構成。gRPC Reflection による動的サービス探索に対応。Middleware/Interceptor による **OIDC 認証** のガバナンスを一元化。
- **Domain**: 唯一の **Source of Truth**。インフラの制約から独立した **不変条件(Invariants)** を抽象インターフェースとして定義。全レイヤーの依存が向かう「不動の頂点」として配置し、コアロジックの再利用性を最大化。
- **Usecase**: `errgroup` を活用した **Fail-fast な並行処理制御** を実装。具象実装を一切参照せず、純粋なビジネスロジック（並行アップロードのオーケストレーション等）に特化。
- **Infrastructure & Elastic Persistence**: Factory Pattern により、ストレージ (GCS/Local/S3(予定)) やデータベース（Cloud SQL/In-Memory）を動的に切り替える **Plug-and-Play** な構成。今後予定している Redis による分散ロック（冪等性制御）や流量制限 も、既存ロジックに影響を与えずこのレイヤーのアダプター追加のみで対応可能。
- **Ingest Pipeline (Event-Driven Integration)**: ファイルのライフサイクル変更を契機とする非同期通知基盤。DIP に基づく抽象化により、Python RAG基盤等の外部システムと疎結合に連携。ドメイン層に影響を与えず、 Pub/Sub(予定) を用いたイベント駆動型メッセージングモデルへシームレスに移行可能な構造を実証。
- **Observability & Lifecycle**: cmd による DI（依存注入）制御により、API/gRPC/CLI の実行形態を柔軟に選択可能。`slog` 拡張により REST/gRPC のプロトコル境界や並行処理の壁を跨いで `context` 内の Trace ID を伝播させ、リクエスト単位の完全なログ追跡（分散トレーシングの布石）を確立。

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

## 🗺 プラットフォームアーキテクチャ・ロードマップ

本基盤の市場価値とエンタープライズシフトを高めるため、「基盤堅牢化（非機能要件・ガバナンス）」と「基盤拡張（機能要件・エコシステム）」の2大レイヤーに分類し、アーキテクチャの検証および拡充を段階的に推進しています。

🛡️ 1. 基盤堅牢化（Platform Hardening & Governance）
システムの信頼性、セキュリティ、ガバナンスを最大化するための非機能要件レイヤー。

📂 Schema-First & API Governance

- [x] **スキーマ駆動開発 (SSOT)** 🎉 *Done*
  - **Single Source of Truth**: OpenAPI 3.0 および Protocol Buffers を採用。仕様と実装の乖離を構造的に排除。
  - **自動生成の統合**: `oapi-codegen` および `protoc` による型安全な Go/Python コードおよび型定義（.pyi）の自動生成を Makefile で完全パイプライン化。
  - **gRPC Reflection の導入**: サービス定義を動的に公開。`grpcurl` 等による高速なローカルデバッグサイクルを確立。

🏗 Architecture & Reliability

- [x] **マルチプロトコル・アダプターの構築** 🎉 *Done*
  - **Handler Layer の分離**: 同一の Interactor (Usecase) を Echo (REST) と gRPC の両ハンドラで共用。プロトコル非依存の純粋なビジネスロジックを隔離。
  - **クリーンな入力マッピング**: HTTP クエリパラメータを Domain 層の抽象型（Specification）へ安全にマッピングする玄関口を実装。`limit/offset` によるページネーションバリデーションを全レイヤーで統合。

- [x] **高並行処理における安全なスレッド制御** 🎉 *Done*
  - **並行処理の Fail-fast**: `errgroup` を用いた並行ゴールーチンの制御により、エラー発生時の迅速なキャンセル伝播とメモリリーク防止を徹底。
  - **Context駆動のライフサイクル管理**: タイムアウトやシグナルキャンセルをコンテキスト経由で全並行タスクへ確実に伝播させる堅牢な実行基盤を構築。

- [x] **エンドツーエンド・オブザーバビリティ (Distributed Tracing)** 🎉 *Done*
  - **Trace ID 伝播**: HTTP Middleware / gRPC Interceptor のプロトコル境界を跨いで  `context` 内の Trace ID を伝播。
  - **言語間トレーシングの同期**: `slog` ハンドラを拡張し、Go ↔ Python（gRPC メタデータ経由）の言語間通信を跨いだ、リクエスト単位の完全な分散トレーシングの土台を確立。

- [x] **非同期タスクのデータモデリングと状態管理** 🎉 *Done*
  - **非同期ステータス遷移の厳密化**: `FileMetadata` に `Status` (Pending/Processing/Completed/Failed) を導入。冪等性を意識した状態遷移モデルにより、非同期なデータ処理状態の追跡を可能に。
  - **データプロバナンス（由来）の保持**: スキーマレベルで `Source` フィールドを定義し、分散パイプラインを流れるデータの「生成元トレーサビリティ」を構造的に担保。

- [x] **Polyglot Architecture (言語間コンテナ連携)** 🎉 *Done*
  - **マルチ言語構成**: Docker Compose ネットワーク内で Go（API）と Python（RAG Worker）を 完全統合。Python 側は Webhook 受付用の FastAPI（HTTP）と、完全非同期な `grpc.aio`（gRPC）の2つのコンテナへ完全に役割を分離・自走化。

📂 Database & Persistence Strategy

- [x] **DB 永続化とトランザクション整合性の管理** 🎉 *Done*
  - **高度な検索**: **Specification Pattern** を導入。PostgreSQL の**配列演算子・GINインデックス**による高速なフィルタリングおよびプレースホルダによる動的クエリ構築を実現。
  - **整合性担保**: **補償トランザクション**を実装し、DB保存失敗時の GCS ロールバックを自動化。`pgxpool` と Unix ドメインソケットを用いたセキュアな接続基盤を構築。

- [x] **DBマイグレーションの自動化** 🎉 *Done*
  - **指数バックオフによる接続リトライ**: DBコンテナの起動遅延を許容するリトライ戦略を実装。
  - **Single Binary Strategy**: `golang-migrate` と `io/fs`(embed) を活用し、バイナリ内包型の自動マイグレーションを実現。環境差分による不具合を**仕組みで排除**。

⚙️ CI/CD & Cloud Native

- [x] **Architecture Testing (理論駆動の実証)** 🎉 *Done*
  - **静的解析の強制**: `go-arch-lint` により **DIP (依存性逆転の原則)** を静的に強制し、レイヤー間の密結合と設計の腐敗を自動で遮断。
  - **品質管理の自動化**: `golangci-lint`（ `errcheck / staticcheck`等）による商用グレードの Go 品質管理と、CI パイプラインでの機密情報混入の静的検知を統合。

- [x] **GitHub Actions による高度な CI 構築** 🎉 *Done*
  - **Python 品質管理の一元化**: `Ruff` による高速な Linter/Formatter 検証、および `Mypy` による厳格な静的型チェックを単一パイプラインへ集約。
  - **運用最適化**: `workflow_dispatch` を導入し、コストや状況に応じた**柔軟な手動デプロイ制御(If-conditional flow)** を確立。

- [x] **Cloud Run への自動デプロイ (CD)** 🎉 *Done*
  - **Attack Surface 最小化**: **Distroless** イメージを採用し、実行環境の脆弱性リスクを根本から低減。
  - **Credential Zero (OIDC)**: Artifact Registry 連携と **Workload Identity** による **Keyless 認証 (OIDC)** を確立。サービスアカウントキーの管理・内包を完全に排除。

- [x] **IaC 化 (Terraform)による再現性の確保** 🎉 *Done*
  - **構成同期（Drift Detection）**: 既存リソースの状態をコードへ正確に反映。**Drift（環境差分）を完全に解消**し、コードと実環境の完全な同期を完遂。
  - **最小権限の原則 (Least Privilege)**: サービスアカウントに対し、実行時（Object Admin）と管理時（Storage Admin）の権限を分離。セキュアな IAM 設計を実証。
  - **リソースの廃棄・再作成制御**: `force_destroy` 等の属性定義により、リソースの廃棄・再作成プロセスを宣言的に記述。

⏩ 次期フェーズ：非機能・ガバナンスの深化
- [ ] **インフラ制約のデータ抽象化（Policy-Driven Architecture）**: テナントごとのアクセス制限ルールをテーブル化し、ミドルウェアで動的に評価。WAF等の外部インフラ設定に依存しないポータブルなアクセス制御を標準化。

- [ ] **ゼロメモリ・ペタバイト転送（Signed URL Session）**: コンテナの OOM（メモリ枯渇）を回避するため、バイナリ中継を排除した file_transfer_sessions によるワンタイムトークンのライフサイクル管理設計（ローカル検証用のボリュームマウント共有）を確立。

- [ ] **不変の状態遷移と監査証跡（Audit Trail & AI Provenance）**: 非同期パイプラインにおけるデータの先祖返りを防ぐ厳格なステータスチェック（ENUM型）と、RAG インジェスト時のソース追跡（プロバナンス）を可能にする Trace ID 内包型の改ざん不可能な操作ログ構造を定義。

- [ ] **マルチテナント隔離の完全防衛（IDOR / パス推測の徹底遮断）**: 物理ストレージパスの隠蔽、メタデータ紐付けによるアクセス認可の徹底、および GCS 具象レイヤーにおけるマルチテナント専用バケット/プレフィックス隔離ポリシーの厳格化。

- [ ] **OpenTelemetry による分散トレーシングの高度化**: コンテナネットワークを跨ぐスパン（Span）ベージングの可視化、およびテナントごとのリソース消費（CPU/メモリ/APIコール数）メトリクスの露出。

🚀 2. 基盤拡張（Platform Capabilities & Ecosystem）
アプリケーションとしての価値を高める機能要件、および外部エコシステムとの連携レイヤー。

🧠 Phase 1: RAG 拡張 & イベント駆動コア [RAG]
- [x] **ドメイン駆動 RAG 抽出パイプラインのコア実装** 🎉 *Done*
  - **ドメインイベントの定義**: ファイル保存をトリガーとした「テキスト抽出 → チャンク分割 → ベクトル化」の一連のフローを不変なドメインイベントとして定義。
  - **ポリモーフィズム設計**: 拡張子を自動判定して適切なパーサーを動的選択する `ExtractorFactory` によるポリモーフィズム設計と、共通設定に基づく `Chunker` を組み合わせた非同期データ加工パイプラインのコアを確立。

💬 Phase 2:メッセージング拡張 [Pub/Sub 抽象化]
- [ ] **イベント駆動アーキテクチャの導入**
  - **メッセージングモデルの隠蔽**: GCP Pub/Sub や AWS SNS/SQS 等の外部メッセージブローカーに依存しない、Go のインターフェースを用いた抽象メッセージング層の定義。
  - **インフラの差し替え可能性**: ドメインイベントのパブリッシュ/サブスクライブをインフラ層から分離し、ローカル環境（Go Channel）とクラウド環境（Managed Pub/Sub）をシームレスに切り替える疎結合アーキテクチャの実証。

🔒 Phase 3: 分散ロック & 流量制御 [Redis]
- [ ] **Redis による分散ロックと冪等性制御**
  - **二重実行の徹底防止**: 重い非同期バッチ処理や外部通知パイプラインにおいて、Redis を用いた「Idempotency Control (冪等性制御キー)」を実装。
  - **分散ロックのリース管理**: 複数コンテナが同時に同一ファイルを処理するのを防ぐため、ロックの獲得・リース期間の自動延長（Watchdog機構）を組み込んだ安全な並行処理排他制御。
- [ ] **テナントごとの流量制御 (Rate Limiting)**
  - **クォータ管理の導入**: 特定のテナントによる大量リクエストがシステム全体の可用性を阻害しないよう、Redis の Token Bucket / Fixed Window アルゴリズムを用いた、同一テナント内のレートリミット基盤を構築。

🛠 Phase 4: デベロッパーエクスペリエンス [React & CLI]
- [ ] **管理用ダッシュボード (React) & CLI の提供**
  - **進捗のリアルタイム可視化**: 各テナントのファイル処理ステータス、デッドレターキュー（DLQ）に溜まったエラー、RAG インジェストの進捗を視覚的に追跡できる軽量な管理画面（React）の提供。
  - **開発者向け Scaffolding の統合**: 新たなストレージアタッチ（S3/Azure Blob等）や新プロトコルを追加する際、ボイラープレートコードを全自動生成する CLI 開発ツールの提供（Schema-First との連動）。

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
│   │   └── core/          # 共通基盤
│   │       ├── config.py            # 環境変数・設定管理
│   │       └── logger.py            # ログの一元初期化
│   ├── Dockerfile         # Python 実行環境のコンテナ定義
│   ├── pyproject.toml     # Python ツールチェーン・静的解析設定
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

### 1. 依存ツールのセットアップ & コード生成
OpenAPI スキーマから型安全な Go コードを生成します。

```bash
# 依存ライブラリの同期
go mod tidy

# OpenAPI スキーマからハンドラ・型定義を自動生成
make gen-api
```

### 2. 環境の起動
開発スタイルに合わせて、以下のいずれかの方法で起動してください。

```bash
### 🐳 A. Docker Compose (Recommended)
# DBリトライ・オートマイグレーションを含めたフル環境を起動
docker compose up --build

### 💻 B. Go Run (Lightweight Mode)
# 外部インフラに依存せず、インメモリDBとローカルストレージで起動
STORAGE_TYPE=LOCAL DB_TYPE=INMEMORY go run cmd/api/main.go
```

### 3. 動作確認（ヘルスチェック）
```bash
# システムの稼働状況と起動時パフォーマンス（Gain）を確認
curl http://localhost:8080/health
```

### 4. 品質管理・静的解析の実行（商用グレードの検証）
コードを変更した際は、以下のコマンドで Go / Python それぞれの厳格な Linter をローカルで実行できます。

```bash
# Go 側の Linter 実行
golangci-lint run

# Python 側の Linter & 型チェック実行 (Ruff & Mypy)
make lint-python
```

```text
> [!TIP]
**DBリトライ戦略**: DBコンテナの起動遅延を許容する「指数バックオフによるリトライアルゴリズム」を実装済みです。`docker compose` 実行時、依存関係の順序を問わず、サービスは自動で正常な状態に収束します。
```
