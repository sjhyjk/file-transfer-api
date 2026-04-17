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
  credentials = file("../gcp-key.json") # .env の GCP_KEY_FILE と合わせる
}

# --- Storage (GCS) ---

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
  public_access_prevention = "enforced"
}

# RAG 用のデータソースバケット
resource "google_storage_bucket" "rag_source_bucket" {
  name          = "rag-source-${var.project_id}"
  location      = "US-WEST1"
  force_destroy = true

  # RAGの機密データを守るための必須設定
  public_access_prevention = "enforced"
}

# --- Database (Cloud SQL) ---

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
      # 注意: 本来はauthorized_networksで自分のIPを制限するのが安全です
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
