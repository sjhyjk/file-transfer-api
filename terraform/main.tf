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
}
