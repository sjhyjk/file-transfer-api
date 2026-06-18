# terraform/variables.tf

variable "project_id" {
  description = "GCP Project ID"
  type        = string
  default     = "file-transfer-api-project" # 先ほど確認したID
}

variable "bucket_name" {
  type        = string
  default     = "file-transfer-bucket-syou-20240121" # 既存のバケット名
}

variable "db_password" {
  description = "Password for the PostgreSQL app_user"
  type        = string
  sensitive   = true # コンソール出力時にマスクされます
}

variable "pubsub_topic_id" {
  type        = string
  description = "GCP Pub/Sub Topic ID for file ingestion events"
  default     = "file-ingest-topic"
}
