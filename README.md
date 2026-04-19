# Go Parallel File Transfer API (Architecture Study)

![Build Status](https://github.com/sjhyjk/file-transfer-api/actions/workflows/docker-build.yml/badge.svg)

本プロジェクトは、Goの並行処理モデル（Concurrency）とクリーンアーキテクチャ（Clean Architecture）の設計思想の習得を目的とした、クラウドストレージ転送基盤の技術検証リポジトリです。**並行処理によるスループットの最大化**を実証しつつ、**静的解析によるアーキテクチャの強制**を導入することで、インフラの制約に縛られない高度なポータビリティと保守性を両立しています。

## 🚀 実装の柱

### 1. クリーンアーキテクチャによる疎結合設計
依存性の逆転（DIP）を徹底し、ビジネスロジックを特定の実行環境やインフラから分離。さらに go-arch-lint を用いた Architecture Testing と、slog による構造化ログを全層に導入し、本番運用に耐えうる「堅牢さ」と「観測可能性」を確保しています。

- **Domain**: ファイルエンティティとリポジトリの外部通信用（Repository/Pipeline）インターフェースを配置。数学的な「定義の抽象化」を意識し、全レイヤーの依存の頂点として定義。
- **Usecase**: 並行アップロードおよび通知の制御ロジックをカプセル化。`infra` 層の存在を一切知らず、`domain` インターフェースのみを介して動作。
- **Infrastructure (Factory Pattern)**: 環境変数 (`STORAGE_TYPE`) 一つで GCS や S3（予定）の実装を動的に切り替える「交換可能性」を実現。
- **cmd**: アプリケーションの起動（Dependency Injection）を担当するエントリポイント。レイヤー構造の外側に配置することで、実行環境（CLI/API等）の交換可能性を確保。
- **Architecture Enforcement**: go-arch-lint を導入。DIP（依存性逆転の原則）が守られているかを静的に自動検証し、設計の腐敗を物理的に遮断。
- **Structured Logging (slog)**: 全レイヤーでJSON形式の構造化ログを出力。特定のファイル名やDB_IDに基づくログ追跡（分散トレーシングの基礎）を可能にしました。

### 2. Goの並行処理モデル (errgroup による Fail-fast 制御)
GoroutineとChannelに加え、`golang.org/x/sync/errgroup` を導入。単なる並行実行に留まらず、以下の高度な制御を実現しています。
- **Fail-fast 実装**: 複数のアップロードのうち1つでもエラーが発生した場合、Context を通じて他の Goroutine の I/O 処理（GCS通信等）を即座に中断。計算リソースとネットワークコストの浪費を構造的に防ぎます。
- **スループット最適化**: 3ファイル同時アップロードにおいて **1.6s → 0.6s** への高速化（約62%改善）を実証済み。

### 3. コンテナ戦略とセキュリティ
- **Multi-stage Build & Distroless**: 実行バイナリのみを抽出した軽量イメージ（Distroless）により、攻撃面を最小化。
- **Environment Variables**: `.env` による設定の外部注入により、機密情報のハードコードを排除し、商用グレードのリンター警告（`SecretsUsedInArgOrEnv`）をクリア。
- **Security & Auth**: ローカルではサービスアカウントキーを使用し、Cloud Run 上では **ADC (Application Default Credentials)** を活用。ソースコードに認証情報を一切含めない、クラウドネイティブなセキュリティ設計を採用。

### 4. オブザーバビリティ (Observability)
- **Structured Logging (log/slog)**: 全レイヤーで JSON 形式の構造化ログを出力。file_name や db_id などの属性を付与することで、Cloud Logging 等での高度なフィルタリングと原因特定を可能にしました。
- **Context-Aware Design**: ログ出力を含む全プロセスで context.Context を保持。並行処理におけるキャンセレーション制御に加え、分散トレーシング（Trace ID の伝播）に対応可能な土台を構築済みです。

### 5. CI/CD パイプライン (GitHub Actions)
「Docker Desktop に依存しない開発」を実現するため、GitHub Actions によるフルオートメーションを構築。

- **Continuous Integration**: `go test` による自動ユニットテストを実行し、品質が担保されたコードのみをビルド。
- **Continuous Deployment**: Artifact Registry への自動ビルド・プッシュ、および **Cloud Run への自動デプロイ** を実現。
- **Vulnerability Scanning**: Artifact Analysis を有効化し、OS/パッケージレベルの脆弱性を自動検知するセキュアなサプライチェーンを構築。

### 6. データベース永続化とステート管理
Cloud SQL (PostgreSQL) を導入し、ストレージへの物理保存と同期して、ファイル名・サイズ・ステータス等のメタデータを永続化。
- **Schema as Code**: マイグレーションロジックをインフラ層に、SQL資産をルートの migrations/ に配置。iofs ドライバを介した依存性注入（DI）により、アプリケーションとDBスキーマのライフサイクルを安全に同期させています。

### 🔌 接続アーキテクチャの最適化
- **Unixドメインソケット接続**: Cloud RunからCloud SQLへの接続には、パブリックIP経由ではなく `/cloudsql/` ディレクトリを介したUnixドメインソケットを採用。高速かつセキュアな通信経路を確保し、ネットワークオーバーヘッドを最小化。
- **コネクションプールの最適化**: `pgxpool` を活用し、並行処理下での効率的な接続管理を実現。
- **Secret管理の徹底**: パスワード等の機密情報は GitHub Secrets 経由でデプロイ時に動的に注入。ソースコードやイメージ内への機密情報の混入を完全に排除。
- **抽象化によるポータビリティ**: インターフェースによる実装の隠蔽を行い、DB実装（PostgreSQL等）の交換可能性を担保。

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

#### プロフェッショナル・ベンチマーク (Go standard `testing.B` による実測)
モック環境下での10ファイル同時処理コスト（530回の試行平均）:
- **Average Latency**: **2.56 ms / op**
- **Memory Efficiency**: **1,427 B / op** (極めて低メモリな実行を実現)
- **Allocations**: **43 allocs / op**

**考察**: 10並列の処理においてもオーバーヘッドは 3ms 未満に抑えられており、大規模なデータ転送基盤としてのスケーラビリティを定量的に実証済みです。

### 考察と設計判断
本検証により、Go の軽量スレッド（Goroutine）を活用することで、インフラ構成を変更することなく I/O ボトルネックを大幅に解消できることを実証しました。このデータは、将来的な大規模データ転送を伴うドメイン（データインジェスト基盤等）において、リソースコストを維持したままスループットを向上させるための重要な「設計判断材料」となります。

## ☁️ Infrastructure as Code (Terraform)

本プロジェクトでは、クラウド資源（GCSバケット・IAM権限）を Terraform によりコード管理しています。

- **Drift Detection**: 手動構築された既存リソースを `terraform import` により管理下へ移行。設計値（Tokyo）と実態（US-West）の乖離を検知し、データ保護の観点から構成定義を修正・同期。
- **Least Privilege (IAM)**: サービスアカウントに対し、実行時（Object Admin）と管理時（Storage Admin）で権限を分離。最小権限の原則に基づいた安全な IaC 運用を実証。
- **Lifecycle Management**: `force_destroy` 等の属性定義により、リソースの廃棄・再作成プロセスを宣言的に記述。

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

## 🛠 今後の検証ロードマップ

- [x] **IaC 化 (Terraform)** 🎉 *Done*
  - バケット・IAM のコード管理を実施。既存リソースの `import` と構成同期を完了。

- [x] **GitHub Actions による CI 構築** 🎉 *Done*
  - **自動テスト**: `go test` によるユニットテストを全プッシュ時に実行し、デグレードを防止。
  - **ベンチマーク自動化**: パフォーマンス特性を `testing.B` で常時監視。
  - **セキュアビルド**: Docker マルチステージビルドによる軽量化と、Artifact Analysis による脆弱性スキャンを統合。

- [x] **Cloud Run への自動デプロイ (CD)** 🎉 *Done*
  - GitHub Actions を通じて Artifact Registry へイメージをプッシュし、Cloud Run へシームレスにデプロイ。環境変数（`BUCKET_NAME`等）の外部注入による「ポータブルな実行環境」を実現。

- [x] **DB 永続化とトランザクション整合性の管理** 🎉 *Done*
  - Cloud Run から Unix ドメインソケット経由で Cloud SQL (PostgreSQL) へ接続。
  - RETURNING 句による ID の即時取得など、効率的な実装を完了。
  - pgx を活用したコネクションプールの最適化により、並行処理下での安定性を確保。

- [x] **Architecture Testing (理論駆動の実証)** 🎉 *Done*
  - `go-arch-lint` を導入し、依存関係の静的解析テストを CI に統合。
  - **DIP (依存性逆転の原則) の強制**: `main.go` から `usecase` への注入時にインターフェースを介することをルール化し、`infra` が `usecase` を汚染しない設計を数学的に担保。

- [x] **オブザーバビリティ (運用の透明性) の標準化** 🎉 *Done*
  - `log/slog` による構造化ログへの全面移行。
  - 属性ベースのログ出力により、原因特定が容易な「守りの運用」を実現。
  - 並行処理のコンテキストを保持し、トレーサビリティを確保。

- [x] **DBマイグレーションの自動化 (実務レベルの運用)** 🎉 *Done*
  - `golang-migrate` と Go の `embed` 機能を統合。
  - アプリケーション起動時にスキーマを自動同期する仕組みを構築。
  - **Single Binary Strategy**: SQLファイルをバイナリに内包することで、コンテナデプロイ時の資材漏れを物理的に防ぎ、ポータビリティを極限まで高めました。

- [x] **堅牢なエラーハンドリング (errgroup による Fail-fast 実装)** 🎉 *Done*
  - `golang.org/x/sync/errgroup` を導入し、並行処理中のエラー伝播を型安全に実装。
  - 1つのエラー発生時に `context.Context` を通じて他の処理を即座にキャンセルする Fail-fast 制御により、クラウドのリソース浪費を防止。

- [ ] **データ取得用 API の実装**
  - 保存されたメタデータを一覧取得・フィルタリングする Read 系 API の実装。

- [ ] **gRPC / スキーマ駆動によるシステム間連携**
  - Protocol Buffers による型定義を先行させ、マイクロサービス化を見据えた高性能・型安全な内部通信の検証。

- [ ] **RAG / データインジェスト基盤への統合**
  - 本基盤を前処理パイプラインとして活用し、イベント駆動（Pub/Sub）による非同期なデータ処理連鎖の実装。 

## 📁 プロジェクト構造
```text
.
├── cmd/                # Entry Point (実行環境の決定・DI・起動)
│   └── api/
│       └── main.go     # DIを行い、Usecaseを起動
├── internal/           # Business Logic (クリーンアーキテクチャのコア)
│   ├── domain/         # Entity & Repository Interface (DIPの起点)
│   │   ├── file.go        # ファイルの実体（Entity）
│   │   ├── repository.go  # 保存(Repo)と通知(Pipeline)の定義
│   │   └── metadata.go  # RAG連携用の属性定義
│   ├── usecase/        # Business Logic (並行処理・制御フロー)
│   └── infra/          # Infrastructure Adapters (技術的詳細の実装)
│       ├── storage_factory.go  # インフラ切り替えの司令塔
│       ├── gcs/                # GCS 具象実装
│       |   └── repository.go
│       └── repository/    # 永続化層の具象実装
│           └── sql/       # Cloud SQL (PostgreSQL) 永続化・マイグレーション
│               ├── db.go         # コネクション・CRUD実装
│               └── migrations.go # golang-migrate 実行ロジック
├── migrations/         # DB スキーマ管理 (SQLファイル)
│   ├── 000001_create_files_table.up.sql
│   └── 000001_create_files_table.down.sql
├── terraform/          # Infrastructure as Code (GCPリソース定義)
│   ├── main.tf         # GCSリソース・Provider定義
│   └── variables.tf    # プロジェクトID・バケット名の変数管理
├── .github/workflows/  # CI/CD パイプライン (GitHub Actions)
│   └── docker-build.yml # 自動コンテナビルド定義
├── archives/           # 開発初期の実装や試行錯誤の軌跡 (ビルド対象外)
├── assets.go           # ★ プロジェクト共通資産（SQL等）の embed 定義
├── Dockerfile          # マルチステージビルドによる軽量実行イメージ定義
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
      ▼ [HTTPS/JSON]
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

本プロジェクトはクリーンアーキテクチャに基づき、依存方向を外側から内側（Domain）へ一方向に制限しています。
この制約は `go-arch-lint` によって静的に強制されており、設計意図に反するインポートは CI で遮断されます。



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

domain 層を全パッケージの「最小単位」として定義し、すべての外部依存（DB/GCS）をこの抽象に紐付けることで、ビジネスロジックの純粋性を担保しています。これは単なる規約ではなく、go-arch-lint による CI 落ちを伴う制約です。

※ go-arch-lint により、`usecase` が `infra` の具象パッケージを
  直接インポートすることを禁止し、DIP（依存性逆転の原則）を担保しています。
```
