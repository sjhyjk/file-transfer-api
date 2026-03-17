# Go Parallel File Transfer API (Clean Architecture)

数学的思考に基づき、保守性とスケーラビリティを追求したファイル転送基盤です。
単なる実装に留まらず、設計の抽象化と並行処理の最適化を行っています。

## 🛠 進化のプロセス
本リポジトリには、開発の過程をあえて残しています。
- `archives/main_botu.go`: 初期実装（1枚岩のメインファイル）
- `cmd/api/main.go`: 現在の実装（クリーンアーキテクチャ採用、並行処理最適化済み）

## 🚀 特徴
- **クリーンアーキテクチャ**: 依存性の逆転（DIP）により、インフラ（GCS）とビジネスロジックを分離。
- **並行処理 (Goroutine)**: 3ファイルの同時アップロードにより、処理時間を約2.1s → 0.9sへ短縮（約50%改善）。
- **Docker/Distroless**: セキュリティと軽量化を両立したマルチステージビルド。
- **環境変数の外部化**: `.env` による設定管理で、セキュリティと可搬性を確保。

## 🏗 アーキテクチャ構成


- **Domain**: エンティティとリポジトリのIF定義
- **Usecase**: 並行アップロードの制御
- **Infrastructure**: GCS SDKを用いた具象実装

## 🏃 実行方法
1. `.env` を作成し、`BUCKET_NAME` 等を設定。
2. `docker build -t file-api .`
3. `docker run --rm --env-file .env file-api`
