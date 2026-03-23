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
