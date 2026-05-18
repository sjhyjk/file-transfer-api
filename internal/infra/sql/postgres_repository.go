// internal/infra/sql/postgres_repository.go

package sql

import (
	"github.com/jackc/pgx/v5/pgxpool"
)

// DB 接続を保持する構造体
// Repository は PostgreSQL へのデータアクセスをカプセル化します
// Repository 構造体の名前自体はシンプルに維持（外部からは sql.Repository として見える）
type Repository struct {
	Pool *pgxpool.Pool
}

// データベースへの接続を開始する
// NewRepository はリポジトリのインスタンスを生成します
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{Pool: pool}
}

// Close は接続プールを安全に閉じます
func (r *Repository) Close() {
	if r.Pool != nil {
		r.Pool.Close()
	}
}
