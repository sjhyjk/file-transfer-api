# terraform/main.tf

terraform {
  required_version = ">= 1.0.0"
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 5.0"
    }
  }
}

provider "google" {
  project = var.project_id # variables.tf で定義する変数
  region  = "us-west1"
}

# ---------------------------------------------------------------------------
# 📦 Storage (GCS)
# ---------------------------------------------------------------------------

# 既存のバケットを管理対象にするための定義
resource "google_storage_bucket" "file_transfer_bucket" {
  name          = var.bucket_name
  location      = "US-WEST1"
  force_destroy = true # 削除時に中身があっても消せる設定（検証用）

  uniform_bucket_level_access = true # セキュリティのベストプラクティス
  
  # 誤って公開されないための設定
  public_access_prevention = "enforced"
}

# Python 比較検証用のバケット
resource "google_storage_bucket" "python_test_bucket" {
  name          = "python-bench-bucket-${var.project_id}"
  location      = "US-WEST1"
  force_destroy = true
  uniform_bucket_level_access = true
  public_access_prevention = "enforced"
}

# RAG 用のデータソースバケット
resource "google_storage_bucket" "rag_source_bucket" {
  name          = "rag-source-${var.project_id}"
  location      = "US-WEST1"
  force_destroy = true
  uniform_bucket_level_access = true

  # RAGの機密データを守るための必須設定
  public_access_prevention = "enforced"
}

# ---------------------------------------------------------------------------
# 🔐 IAM & Service Account (Runtime & CI/CD)
# ---------------------------------------------------------------------------

# Cloud Run 実行専用のサービスアカウント (Runtime SA)
resource "google_service_account" "app_runtime_sa" {
  account_id   = "file-transfer-app-runtime"
  display_name = "Cloud Run App Runtime Service Account"
}

# メインのバケットへのオブジェクト操作権限付与
resource "google_storage_bucket_iam_member" "main_bucket_access" {
  bucket = google_storage_bucket.file_transfer_bucket.name
  role   = "roles/storage.objectAdmin" # 読み書き可能、バケット削除は不可
  member = "serviceAccount:${google_service_account.app_runtime_sa.email}"
}

# RAGソースバケットへの「読み取り専用」権限（さらに絞る例）
resource "google_storage_bucket_iam_member" "rag_bucket_viewer" {
  bucket = google_storage_bucket.rag_source_bucket.name
  role   = "roles/storage.objectViewer" # 読み取りのみ
  member = "serviceAccount:${google_service_account.app_runtime_sa.email}"
}

# GitHub Actions がデプロイとプッシュを行うための権限付与群
# サービスアカウント自身が自分を Cloud Run に割り当てるための権限
resource "google_project_iam_member" "sa_user" {
  project = var.project_id
  role    = "roles/iam.serviceAccountUser"
  member  = "serviceAccount:${google_service_account.app_runtime_sa.email}"
}

# Artifact Registry への書き込み（Push）権限
resource "google_project_iam_member" "artifact_registry_writer" {
  project = var.project_id
  role    = "roles/artifactregistry.writer"
  member  = "serviceAccount:${google_service_account.app_runtime_sa.email}"
}

# Cloud Run の管理者権限（デプロイ実行に必要）
resource "google_project_iam_member" "cloudrun_developer" {
  project = var.project_id
  role    = "roles/run.developer"
  member  = "serviceAccount:${google_service_account.app_runtime_sa.email}"
}

# ---------------------------------------------------------------------------
# 🗄️ Database (Cloud SQL)
# ---------------------------------------------------------------------------

/*
# Cloud SQL への接続権限 (Cloud SQL クライアント)
resource "google_project_iam_member" "sql_client" {
  project = var.project_id
  role    = "roles/cloudsql.client"
  member  = "serviceAccount:${google_service_account.app_runtime_sa.email}"
}

# Cloud SQL インスタンス
resource "google_sql_database_instance" "postgres" {
  name             = "file-transfer-db"
  database_version = "POSTGRES_15"
  region           = "us-west1" # ストレージとリージョンを合わせる

  settings {
    tier = "db-f1-micro" # 開発・検証用の最小インスタンス
    
    backup_configuration {
      enabled = true
    }

    # 最初は接続を容易にするためにパブリックIPを許可（後で閉域網化を検討）
    ip_configuration {
      ipv4_enabled = true

      # 以前接続できていた設定を、そのままコードとして定義します
      # これにより「過去に接続実績があること」と「現在の設計意図」を両立させます
      authorized_networks {
        name  = "home"
        value = "61.23.155.152" # 以前成功した時のIP
      }
    }
  }

  deletion_protection = false # 検証用のため削除保護はオフ
}

# アプリ用データベース
resource "google_sql_database" "database" {
  name     = "transfer_metadata"
  instance = google_sql_database_instance.postgres.name
}

# アプリ用ユーザー
resource "google_sql_user" "db_user" {
  name     = "app_user"
  instance = google_sql_database_instance.postgres.name
  password = var.db_password # variables.tf で定義が必要
}
*/
# ==========================================
# Workload Identity 設定 (GitHub Actions 用)
# ==========================================

# 1. Workload Identity Pool の作成
resource "google_iam_workload_identity_pool" "github_pool" {
  workload_identity_pool_id = "github-actions-pool"
  display_name              = "GitHub Actions Pool"
  description               = "Identity pool for GitHub Actions"
}

# 2. Workload Identity Provider の作成
resource "google_iam_workload_identity_pool_provider" "github_provider" {
  workload_identity_pool_id          = google_iam_workload_identity_pool.github_pool.workload_identity_pool_id
  workload_identity_pool_provider_id = "github-provider"
  display_name                       = "GitHub Actions Provider"

  attribute_mapping = {
    "google.subject"       = "assertion.sub"
    "attribute.repository" = "assertion.repository"
    "attribute.owner"      = "assertion.repository_owner"
    "attribute.refs"       = "assertion.ref"
    "attribute.actor"      = "assertion.actor"
    "attribute.aud"        = "assertion.aud"
  }
  
  # 特定のリポジトリ以外からのアクセスを入り口で弾く設定
  # これにより、GCP側が求める「Claimsの参照」を完全に満たします。
  attribute_condition = "assertion.repository == 'sjhyjk/file-transfer-api'"

  oidc {
    issuer_uri = "https://token.actions.githubusercontent.com"
  }
}

# 3. サービスアカウントに GitHub Actions からの「なりすわり」権限を付与
# ※既存の google_service_account.app_runtime_sa を利用します
resource "google_service_account_iam_member" "workload_identity_user" {
  service_account_id = google_service_account.app_runtime_sa.name
  role               = "roles/iam.workloadIdentityUser"
  member             = "principalSet://iam.googleapis.com/${google_iam_workload_identity_pool.github_pool.name}/attribute.repository/sjhyjk/file-transfer-api"
}

# ==========================================
# Pub/Sub メッセージング基盤
# ==========================================

# 1. Goアプリがイベントをパブリッシュするトピック
resource "google_pubsub_topic" "file_ingest_topic" {
  name = var.pubsub_topic_id

  labels = {
    environment = "production"
    managed_by  = "terraform"
  }
}

# 2. Pythonワーカーがメッセージをプルするサブスクリプション
resource "google_pubsub_subscription" "file_ingest_sub" {
  name  = "file-ingest-sub"
  topic = google_pubsub_topic.file_ingest_topic.name

  # 確認応答（Ack）の締め切り：RAGのパース処理時間を考慮して少し長めの60秒に設定
  ack_deadline_seconds = 60

  # メッセージの保持期間（デフォルト7日間）
  message_retention_duration = "604800s"

  # 冪等性を担保するためのリトライポリシー（必要に応じて調整）
  retry_policy {
    minimum_backoff = "10s"
    maximum_backoff = "600s"
  }
}

# 3. 権限付与：Goアプリ（Runtime SA）がトピックに「発行」できる権限
resource "google_pubsub_topic_iam_member" "go_app_publisher" {
  topic  = google_pubsub_topic.file_ingest_topic.name
  role   = "roles/pubsub.publisher"
  member = "serviceAccount:${google_service_account.app_runtime_sa.email}"
}

# 4. 権限付与：Pythonワーカー（Runtime SA）がサブスクリプションから「購読」できる権限
resource "google_pubsub_subscription_iam_member" "python_worker_subscriber" {
  subscription = google_pubsub_subscription.file_ingest_sub.name
  role         = "roles/pubsub.subscriber"
  member       = "serviceAccount:${google_service_account.app_runtime_sa.email}"
}
