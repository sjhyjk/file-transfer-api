# Go Parallel File Transfer API (Architecture Study)

本プロジェクトは、Goの並行処理モデル（Concurrency）とクリーンアーキテクチャ（Clean Architecture）の設計思想の習得を目的とした、クラウドストレージ転送基盤の技術検証リポジトリです。

## 🚀 実装の柱

### 1. クリーンアーキテクチャによる疎結合設計
依存性の逆転（DIP）を徹底し、ビジネスロジックを特定の実行環境やインフラから分離しています。

- **Domain**: 数学的な「定義の抽象化」を意識し、ファイルエンティティとリポジトリのインターフェースをここに配置し、全レイヤーの依存の頂点として定義。
- **Usecase**: 並行アップロードの制御ロジックをカプセル化。
- **Infrastructure (Factory Pattern)**: 環境変数 (`STORAGE_TYPE`) 一つで GCS や S3（予定）の実装を動的に切り替える「交換可能性」を実現。
- **cmd**: アプリケーションの起動（Dependency Injection）を担当するエントリポイント。レイヤー構造の外側に配置することで、実行環境（CLI/API等）の交換可能性を確保。

### 2. Goの並行処理モデル
GoroutineとChannelを活用し、ネットワークI/Oの待機時間を最適化。3ファイル同時アップロードにおいて **2.1s → 0.9s** への高速化（約53%改善）を実証済みです。

### 3. コンテナ戦略とセキュリティ
- **Multi-stage Build & Distroless**: 実行バイナリのみを抽出した軽量イメージ（Distroless）により、攻撃面を最小化。
- **Environment Variables**: `.env` による設定の外部注入により、機密情報のハードコードを排除し、商用グレードのリンター警告（`SecretsUsedInArgOrEnv`）をクリア。

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

### 考察と設計判断
本検証により、Go の軽量スレッド（Goroutine）を活用することで、インフラ構成を変更することなく I/O ボトルネックを大幅に解消できることを実証しました。このデータは、将来的な大規模データ転送を伴う RAG 基盤等において、リソースコストを維持したままスループットを向上させるための重要な「設計判断材料」となります。

## ☁️ Infrastructure as Code (Terraform)

本プロジェクトでは、クラウド資源（GCSバケット・IAM権限）を Terraform によりコード管理しています。

- **Drift Detection**: 手動構築された既存リソースを `terraform import` により管理下へ移行。設計値（Tokyo）と実態（US-West）の乖離を検知し、データ保護の観点から構成定義を修正・同期。
- **Least Privilege (IAM)**: サービスアカウントに対し、実行時（Object Admin）と管理時（Storage Admin）で権限を分離。最小権限の原則に基づいた安全な IaC 運用を実証。
- **Lifecycle Management**: `force_destroy` 等の属性定義により、リソースの廃棄・再作成プロセスを宣言的に記述。

## 🛠 今後の検証ロードマップ

- [ ] **Python との比較**
  - Python (AsyncIO) と Go (Goroutine) のランタイム特性・メモリ効率・速度を定量比較

- [ ] **AWS S3 対応**
  - Infrastructure 層の差し替えによるマルチクラウド対応

- [x] **IaC 化 (Terraform)** 🎉 *Done*
  - バケット・IAM のコード管理を実施 

- [ ] **RAG パイプライン統合**  
  - 本基盤をデータインジェストの前処理パイプラインとして活用   

## 📁 プロジェクト構造
```text
.
├── cmd/                # Entry Point (実行環境の決定・外部との接点)
│   └── api/
│       └── main.go     # DIを行い、Usecaseを起動
├── internal/           # Business Logic (クリーンアーキ本体)
│   ├── domain/         # Entity & Repository Interface (DIPの起点)
│   │   ├── file.go
│   │   └── repository.go
│   ├── usecase/        # Business Logic (Parallel/Serial Control)
│   └── infra/          # Infrastructure Adapters
│       ├── storage_factory.go  # ★ インフラ切り替えの司令塔
│       └── gcs/                # GCS 具象実装
│           └── repository.go
├── terraform/          # Infrastructure as Code (IaC)
│   ├── main.tf         # GCSリソース・Provider定義
│   └── variables.tf    # プロジェクトID・バケット名の変数管理
├── archives/           # 試行錯誤の軌跡（初期の直列・Monolith実装を保存）
├── Dockerfile          # マルチステージビルド定義
└── .env                # 環境設定（Git管理対象外）

