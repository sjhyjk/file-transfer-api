-- 既存のtagsカラムに対してGINインデックスを作成
CREATE INDEX IF NOT EXISTS idx_file_metadata_tags ON file_metadata USING GIN (tags);
