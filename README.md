# Go Parallel File Transfer API (Architecture Study)

本プロジェクトは、Goの並行処理モデル（Concurrency）とクリーンアーキテクチャ（Clean Architecture）の設計思想の習得を目的とした、クラウドストレージ転送基盤の技術検証リポジトリです。

## 🚀 実装の柱

### 1. クリーンアーキテクチャによる疎結合設計
依存性の逆転（DIP）を徹底し、ビジネスロジックを特定の実行環境やインフラから分離しています。

- **Domain**: 数学的な「定義の抽象化」を意識し、ファイルエンティティとリポジトリのインターフェースを定義。
- **Usecase**: 並行アップロードの制御ロジックをカプセル化。
- **Infrastructure**: GCS SDKを用いた具体的な具象実装を担当。
- **cmd**: アプリケーションの起動（Dependency Injection）を担当するエントリポイント。レイヤー構造の外側に配置することで、実行環境（CLI/API等）の交換可能性を確保。

### 2. Goの並行処理モデル
GoroutineとChannelを活用し、ネットワークI/Oの待機時間を最適化。3ファイル同時アップロードにおいて **2.1s → 0.9s** への高速化（約53%改善）を実証済みです。

### 3. コンテナ戦略とセキュリティ
- **Multi-stage Build & Distroless**: 実行バイナリのみを抽出した軽量イメージ（Distroless）により、攻撃面を最小化。
- **Environment Variables**: `.env` による設定の外部注入により、機密情報のハードコードを排除し、商用グレードのリンター警告（`SecretsUsedInArgOrEnv`）をクリア。

## ⚡ Go の並行処理モデル

Goroutine と Channel によるネットワーク I/O の待機時間最適化を実施。

| 方式 | 実行時間 | 備考 |
|------|----------|------|
| Serial | ~2.1s | 初期の直列・モノリシック実装 |
| Parallel (Go) | ~0.9s | Goroutine による並行処理最適化 |

**→ 約 53% の高速化を実証**

## 🛠 今後の検証ロードマップ

- **Python との比較**  
  - Python (AsyncIO) と Go (Goroutine) のランタイム特性・メモリ効率・速度を定量比較  

- **AWS S3 対応**  
  - Infrastructure 層の差し替えによるマルチクラウド対応  

- **RAG パイプライン統合**  
  - 本基盤をデータインジェストの前処理パイプラインとして活用  

- **IaC 化 (Terraform)**  
  - バケット・IAM のコード管理を実施 

## 📁 プロジェクト構造
```text
.
├── cmd/                # Entry Point (実行環境の決定・外部との接点)
│   └── api/
│       └── main.go     # DIを行い、Usecaseを起動
├── internal/           # Business Logic (クリーンアーキ本体)
│   ├── domain/         # Interface / Entity
│   ├── usecase/        # Application Logic
│   └── infra/          # Adapter (GCS等)
├── archives/           # 試行錯誤の軌跡（初期の直列・Monolith実装を保存）
├── Dockerfile          # マルチステージビルド定義
└── .env                # 環境設定（Git管理対象外）

