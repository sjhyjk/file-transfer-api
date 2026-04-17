# Go Parallel File Transfer API (Architecture Study)

![Build Status](https://github.com/sjhyjk/file-transfer-api/actions/workflows/docker-build.yml/badge.svg)

本プロジェクトは、Goの並行処理モデル（Concurrency）とクリーンアーキテクチャ（Clean Architecture）の設計思想の習得を目的とした、クラウドストレージ転送基盤の技術検証リポジトリです。**「インフラの制約がビジネスロジックを縛らない」ポータブルな設計**と、並行処理によるスループットの最大化を両立しています。

## 🚀 実装の柱

### 1. クリーンアーキテクチャによる疎結合設計
依存性の逆転（DIP）を徹底し、ビジネスロジックを特定の実行環境やインフラから分離しています。

- **Domain**: ファイルエンティティとリポジトリの外部通信用（Repository/Pipeline）インターフェースを配置。数学的な「定義の抽象化」を意識し、全レイヤーの依存の頂点として定義。
- **Usecase**: 並行アップロードおよび通知の制御ロジックをカプセル化。
- **Infrastructure (Factory Pattern)**: 環境変数 (`STORAGE_TYPE`) 一つで GCS や S3（予定）の実装を動的に切り替える「交換可能性」を実現。
- **cmd**: アプリケーションの起動（Dependency Injection）を担当するエントリポイント。レイヤー構造の外側に配置することで、実行環境（CLI/API等）の交換可能性を確保。

### 2. Goの並行処理モデル
GoroutineとChannelを活用し、ネットワークI/Oの待機時間を最適化。3ファイル同時アップロードにおいて **2.1s → 0.9s** への高速化（約53%改善）を実証済みです。

### 3. コンテナ戦略とセキュリティ
- **Multi-stage Build & Distroless**: 実行バイナリのみを抽出した軽量イメージ（Distroless）により、攻撃面を最小化。
- **Environment Variables**: `.env` による設定の外部注入により、機密情報のハードコードを排除し、商用グレードのリンター警告（`SecretsUsedInArgOrEnv`）をクリア。
- **Security & Auth**: ローカルではサービスアカウントキーを使用し、Cloud Run 上では **ADC (Application Default Credentials)** を活用。ソースコードに認証情報を一切含めない、クラウドネイティブなセキュリティ設計を採用。

### 4. CI/CD パイプライン (GitHub Actions)
「Docker Desktop に依存しない開発」を実現するため、GitHub Actions によるフルオートメーションを構築。

- **Continuous Integration**: `go test` による自動ユニットテストを実行し、品質が担保されたコードのみをビルド。
- **Continuous Deployment**: Artifact Registry への自動ビルド・プッシュ、および **Cloud Run への自動デプロイ** を実現。
- **Vulnerability Scanning**: Artifact Analysis を有効化し、OS/パッケージレベルの脆弱性を自動検知するセキュアなサプライチェーンを構築。

### 5. データベース永続化とステート管理
Cloud SQL (PostgreSQL) を導入し、ストレージへの物理保存と同期して、ファイル名・サイズ・ステータス等のメタデータを永続化。

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

- [ ] **DBマイグレーションの自動化 (実務レベルの運用)**
  - `golang-migrate` の統合による、アプリケーション起動時のスキーマ自動同期。

- [ ] **堅牢なエラーハンドリング (異常系の設計実証)**
  - DB失敗時のストレージロールバックや、並行処理中のエラーハンドリングの型化。

- [ ] **データ取得用 API の実装**
  - 保存されたメタデータを一覧取得・フィルタリングする Read 系 API の実装。

- [ ] **オブザーバビリティ (運用の透明性) の標準化**
  - slog による構造化ログの導入と、Trace ID を context で伝播させる分散トレーシングの基礎構築。
  - 基盤側でエラーハンドリングを型化し、開発者が「原因特定しやすい」ログ出力を強制する設計。

- [ ] **Architecture Testing (理論駆動の実証)**
  - go-arch-lint 等を用いた依存関係の静的解析テスト。
  - 「Domain層が外部ライブラリやInfraに依存していないこと」をコードで自動検証し、設計の正しさを数学的に担保。

- [ ] **gRPC / スキーマ駆動によるシステム間連携**
  - Protocol Buffers による型定義を先行させ、マイクロサービス化を見据えた高性能・型安全な内部通信の検証。

- [ ] **RAG / データインジェスト基盤への統合**
  - 本基盤を前処理パイプラインとして活用し、イベント駆動（Pub/Sub）による非同期なデータ処理連鎖の実装。 

## 📁 プロジェクト構造
```text
.
├── .github/workflows/  # CI/CD (GitHub Actions)
│   └── docker-build.yml # 自動コンテナビルド定義
├── cmd/                # Entry Point (実行環境の決定・外部との接点)
│   └── api/
│       └── main.go     # DIを行い、Usecaseを起動
├── internal/           # Business Logic (クリーンアーキ本体)
│   ├── domain/         # Entity & Repository Interface (DIPの起点)
│   │   ├── file.go        # ファイルの実体（Entity）
│   │   ├── repository.go  # 保存(Repo)と通知(Pipeline)の定義
│   │   └── metadata.go  # RAG連携用の属性定義
│   ├── usecase/        # Business Logic (Parallel/Serial Control)
│   └── infra/          # Infrastructure Adapters
│       ├── storage_factory.go  # インフラ切り替えの司令塔
│       ├── gcs/                # GCS 具象実装
│       |   └── repository.go
│       └── repository/    # ★ 永続化層の具象実装
│           └── sql/       # ★ Cloud SQL (Postgres) 実装
│               └── db.go
├── migrations/         # ★ DB スキーマ管理 (golang-migrate)
│   ├── 000001_create_files_table.up.sql
│   └── 000001_create_files_table.down.sql
├── terraform/          # Infrastructure as Code (IaC)
│   ├── main.tf         # GCSリソース・Provider定義
│   └── variables.tf    # プロジェクトID・バケット名の変数管理
├── python_comparison/  # Python (AsyncIO) との性能比較検証用 (In Progress)
├── rag_pipeline/       # RAG インジェスト基盤への拡張用 (In Progress: 設計フェーズ)
├── aws_infrastructure/ # マルチクラウド (S3) 展開用の設計検討用
├── archives/           # 試行錯誤の軌跡（初期の直列・Monolith実装を保存）
├── Dockerfile          # マルチステージビルド定義
└── .env                # 環境設定（Git管理対象外）
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
