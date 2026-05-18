// internal/infra/sql/client.go

package sql

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// OpenWithRetry は指定された URL に対してバックオフを伴う接続プールを確立します
func OpenWithRetry(ctx context.Context, dbURL string) (*pgxpool.Pool, error) {
	slog.InfoContext(ctx, "🔌 Connecting to Cloud SQL (PostgreSQL)...")
	var pool *pgxpool.Pool
	var err error
	maxRetries := 5

	for i := 1; i <= maxRetries; i++ {
		pool, err = pgxpool.New(ctx, dbURL)
		if err == nil {
			// 実際に通信できるか Ping を飛ばして確認
			if err = pool.Ping(ctx); err == nil {
				slog.InfoContext(ctx, "🎉 Database connection pool is ready!")
				return pool, nil
			}
		}

		if pool != nil {
			pool.Close()
		}

		if i < maxRetries {
			slog.WarnContext(ctx, "⚠️ DB not ready. Retrying connection...",
				"attempt", i,
				"max", maxRetries,
				"error", err,
			)
			time.Sleep(2 * time.Second)
		}
	}

	return nil, fmt.Errorf("database pool initialization failed after %d attempts: %w", maxRetries, err)
}
