-- migrations/000001_create_files_table.up.sql

-- 1. ステータス用のENUM型を定義（ドメイン駆動設計の反映）
DO $$ BEGIN
    CREATE TYPE transfer_status AS ENUM ('pending', 'processing', 'completed', 'failed');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

-- 2. メタデータテーブルの作成
CREATE TABLE IF NOT EXISTS file_metadata (
    id BIGSERIAL PRIMARY KEY,
    file_name TEXT NOT NULL,
    file_size BIGINT NOT NULL,
    status transfer_status NOT NULL DEFAULT 'pending',
    source TEXT NOT NULL,
    tags TEXT[], -- PostgreSQLの配列型を活用
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
