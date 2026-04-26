# Go Parallel File Transfer API (Architecture Study)

![Build Status](https://github.com/sjhyjk/file-transfer-api/actions/workflows/docker-build.yml/badge.svg)

本プロジェクトは、Goの並行処理モデル（Concurrency）とクリーンアーキテクチャ（Clean Architecture）の設計思想の習得を目的とした、クラウドストレージ転送基盤の技術検証リポジトリです。**並行処理によるスループットの最大化**を実証しつつ、**静的解析によるアーキテクチャの強制**を導入することで、インフラの制約に縛られない高度なポータビリティと保守性を両立しています。

## 🚀 実装の柱

### 🏛️ クリーンアーキテクチャによる疎結合設計
依存性の逆転（DIP）を徹底し、ビジネスロジックを特定の実行環境やインフラから分離。さらに `go-arch-lint` を用いた **Architecture Testing** 導入することで、設計の腐敗を静的に遮断し、長期的な保守性とポータビリティを担保しています。

- **Domain**: 唯一の **Source of Truth**。数学的な「定義の抽象化」を意識し、外部（Repository/Pipeline）との契約となるインターフェースを配置。全レイヤーの依存が向かう「不動の頂点」として定義。
- **Usecase**: ビジネスロジックの純粋性を維持。`infra` 層の具象実装を一切参照せず、`domain` のインターフェースのみを介して並行アップロードや通知を制御。
- **Infrastructure (Factory Pattern)**: `STORAGE_TYPE` 等の環境変数に基づき、GCS/Local Storage/S3（予定）、Cloud SQL/In-Memory DB を動的に切り替える **Plug-and-Play** な構成を採用。
- **cmd (Main Component)**: 依存注入（DI）と起動のみに特化。アプリケーションを「何として（API/CLI）」動かすかを外部から注入可能にし、コアロジックの再利用性を最大化。
- **Observability**: `slog` による構造化ログを全層に適用。`db_id` 等のコンテキストを伝播させ、分散トレーシングを見据えた「運用の透明性」を確保。

## 🛠 Quick Start (Local Development)

外部インフラ（GCP）に依存せず、ローカル環境のみで API サーバーを即座に起動し、並行処理の挙動を検証可能です。

```bash
# 依存関係なし（In-Memory / Local Storage）で起動
STORAGE_TYPE=LOCAL DB_TYPE=INMEMORY go run cmd/api/main.go
```

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

## 🛠 検証済みロードマップ (Infrastructure & Backend)

📂 Database & Persistence Strategy

- [x] **DB 永続化とトランザクション整合性の管理** 🎉 *Done*
  - **高度な検索実装**: **Specification Pattern** を導入。PostgreSQL の**配列演算子・GINインデックス**による動的かつ高速なフィルタリングを実現。プレースホルダによる動的クエリ構築により、SQLインジェクションを完全に排除。
  - **整合性担保**: **補償トランザクション**を実装し、DB保存失敗時の GCS ロールバックを自動化。`pgxpool` と Unix ドメインソケットを用いたセキュアな接続基盤を構築。
  - **クリーンなAPI設計**: HTTP クエリパラメータを Domain 層の抽象型へ変換する **「マルチプロトコル対応」** の玄関口を実装。`limit/offset` によるページネーションバリデーションを全レイヤー（Handler -> Usecase -> Domain -> Infra）で統合。

- [x] **DBマイグレーションの自動化** 🎉 *Done*
  - **Single Binary Strategy**: `golang-migrate` と `io/fs`(embed) を活用し、バイナリ内包型の自動マイグレーションを実現。環境差分による不具合を**仕組みで排除**。

⚙️ CI/CD & Cloud Native

- [x] **GitHub Actions による高度な CI 構築** 🎉 *Done*
  - **安全性と性能の自動化**: `go test` による自動テスト、`testing.B` による性能監視、および商用グレードのリンターによる**機密情報混入の静的検知**を統合。
  - **運用最適化**: `workflow_dispatch` を導入し、コストや状況に応じた**柔軟な手動デプロイ制御（If-conditional flow）**を確立。

- [x] **Cloud Run への自動デプロイ (CD)** 🎉 *Done*
  - **Attack Surface 最小化**: **Distroless** イメージを採用し、実行環境の脆弱性リスクを根本から低減。
  - **Credential Zero**: Artifact Registry 連携と **Workload Identity** による **Keyless 認証** (ADC活用) を確立し、認証情報のバイナリ内包を完全に排除。

- [x] **IaC 化 (Terraform)による再現性の確保** 🎉 *Done*
  - **構成同期（Drift Detection）**: 既存リソースの状態をコードへ正確に反映。**Drift（環境差分）を完全に解消**し、コードと実環境の完全な同期を完遂。
  - **最小権限の原則 (Least Privilege)**: サービスアカウントに対し、実行時（Object Admin）と管理時（Storage Admin）の権限を分離。セキュアな IAM 設計を実証。
  - **ライフサイクル管理**: `force_destroy` 等の属性定義により、リソースの廃棄・再作成プロセスを宣言的に記述。

🏗 Architecture & Reliability

- [x] **Architecture Testing (理論駆動の実証)** 🎉 *Done*
  - `go-arch-lint` により、`Usecase` が `Infra` に依存しない **DIP (依存性逆転の原則)** を静的に強制。設計の腐敗を自動で遮断する仕組みを構築。

- [x] **オブザーバビリティ & 並行処理制御** 🎉 *Done*
  - `slog` による db_id 付き構造化ログと `errgroup` による **Fail-fast** 制御を実装。単体テストにおいて、並行処理中の一部エラーが全体へ即座に波及・中断される挙動を検証済み。分散トレーシングを見据えた `context` 伝播を 全レイヤーに適用 し、異常検知時の即座な処理中断（リソース浪費防止）を実現。

## 🛠 今後の検証ロードマップ

- [ ] **gRPC / スキーマ駆動によるシステム間連携**
  - `Protocol Buffers` による型定義を先行させ、マイクロサービス化を見据えた高性能・型安全な内部通信の検証。

- [ ] **RAG / データインジェスト基盤への統合**
  - 本基盤を前処理パイプラインとして活用し、イベント駆動（Pub/Sub）による非同期なデータ処理連鎖の実装。 

## 📁 プロジェクト構造
```text
.
├── cmd/                # Entry Point (実行環境の決定・DI・起動)
│   └── api/
│       └── main.go     # DIを行い、Usecaseを起動
├── internal/           # Business Logic (クリーンアーキテクチャのコア)
│   ├── handler/        # 外部接続（HTTPリクエストの解析・レスポンス生成）
│   │   └── file_handler.go
│   ├── domain/         # Entity & Repository Interface (DIPの起点)
│   │   ├── file.go        # ファイルの実体（Entity）
│   │   ├── repository.go  # 保存(Repo)と通知(Pipeline)の定義
│   │   └── metadata.go  # RAG連携用の属性定義
│   ├── usecase/        # Business Logic (並行処理・制御フロー)
│   │   ├── file_interactor.go       # 並行アップロードのコアロジック
│   │   └── file_interactor_test.go  # ロジックの正当性を保証するテスト
│   └── infra/          # Infrastructure Adapters (技術的詳細の実装)
│       ├── factory.go  # インフラ切り替えの司令塔
│       ├── gcs/                # GCS 具象実装
│       |   └── gcs_repository.go
│       ├── local/         # ローカルファイルシステム実装
│       |   └── local_repository.go
│       └── repository/    # 永続化層の具象実装
│           ├── inmemory/  # 高速な検証を可能にするインメモリDB実装
│           │   └── memory_repository.go
│           └── sql/       # Cloud SQL (PostgreSQL) 永続化・マイグレーション
│               ├── db.go         # コネクション・CRUD実装
│               └── migrations.go # golang-migrate 実行ロジック
├── migrations/         # DB スキーマ管理 (SQLファイル)
│   ├── 000001_create_files_table.up.sql
│   ├── 000001_create_files_table.down.sql
│   ├── 000002_add_gin_index_to_tags.up.sql
│   └── 000002_add_gin_index_to_tags.down.sql
├── terraform/          # Infrastructure as Code (GCPリソース定義)
│   ├── main.tf         # GCSリソース・Provider定義
│   ├── outputs.tf      # インフラ出力情報の定義
│   └── variables.tf    # プロジェクトID・バケット名の変数管理
├── .github/workflows/  # CI/CD パイプライン (GitHub Actions)
│   └── docker-build.yml # 自動コンテナビルド定義
├── archives/           # 開発初期の実装や試行錯誤の軌跡 (ビルド対象外)
├── assets.go           # プロジェクト共通資産（SQL等）の embed 定義
├── Dockerfile          # マルチステージビルドによる軽量実行イメージ定義
├── .go-arch-lint.yml   # アーキテクチャの依存関係を強制する定義ファイル
├── go.mod              # 依存関係管理
├── .env                # 環境設定（Git管理対象外）
├── README.md           # 本ドキュメント
│
├── python_comparison/  # [In Progress] Python (AsyncIO) との性能比較検証用
├── rag_pipeline/       # [In Progress] RAG インジェスト基盤への拡張用設計
└── aws_infrastructure/ # [In Progress] マルチクラウド (S3) 展開用の設計検討
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
