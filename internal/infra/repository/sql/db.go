package sql

import (
	"context"
	"file-transfer-api/internal/domain"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DB 接続を保持する構造体
type Repository struct {
	Pool *pgxpool.Pool
}

// データベースへの接続を開始する
func NewRepository(ctx context.Context) (*Repository, error) {
	// 接続URLを環境変数などから組み立てる（先ほどの migrate コマンドで使ったものと同じ）
	// 本来は .env から読み込むべきですが、まずはテスト用に直接指定
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("DATABASE_URL environment variable is not set")
	}

	// pgxpool は「接続のプール」を管理してくれる賢い子です
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database: %w", err)
	}

	// 実際に通信できるか Ping を飛ばして確認
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping failed: %w", err)
	}

	return &Repository{Pool: pool}, nil
}

func (r *Repository) Close() {
	r.Pool.Close()
}

// SaveMetadata はファイル情報を PostgreSQL に保存します
func (r *Repository) SaveMetadata(ctx context.Context, f *domain.FileMetadata) error {
	query := `
        INSERT INTO file_metadata (file_name, file_size, status, source, tags)
        VALUES ($1, $2, $3, $4, $5)
        RETURNING id, created_at;
    `

	// pgx は Go のスライス ([]string) を Postgres の配列 (TEXT[]) としてそのまま扱えます
	err := r.Pool.QueryRow(ctx, query,
		f.FileName,
		f.FileSize,
		f.Status, // 文字列として ENUM にキャストされます
		f.Source,
		f.Tags,
	).Scan(&f.ID, &f.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to insert metadata: %w", err)
	}

	return nil
}

// Create は SaveMetadata と同じ役割として実装します
func (r *Repository) Create(ctx context.Context, record *domain.FileMetadata) error {
	return r.SaveMetadata(ctx, record)
}

// UpdateStatus はステータスを更新します（今回は実装を省略してもエラーは消えます）
func (r *Repository) UpdateStatus(ctx context.Context, id int64, status domain.TransferStatus) error {
	query := `UPDATE file_metadata SET status = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2`
	_, err := r.Pool.Exec(ctx, query, status, id)
	return err
}

// FindByID はIDで検索します（今回は実装を省略してもエラーは消えます）
func (r *Repository) FindByID(ctx context.Context, id int64) (*domain.FileMetadata, error) {
	// 必要になったら実装しましょう。今は一旦 nil を返すだけでもコンパイルは通ります。
	return nil, fmt.Errorf("not implemented")
}
