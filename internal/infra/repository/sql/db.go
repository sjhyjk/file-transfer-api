package sql

import (
	"context"
	"file-transfer-api/internal/domain"
	"fmt"
	"log/slog"
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
// SaveMetadata は domain.MetadataRepository インターフェースの要件を満たす必要があります
func (r *Repository) SaveMetadata(ctx context.Context, f *domain.FileMetadata) error {
	// ★ 追加：レシーバーが nil の場合はエラーを返す（panic防止）
	if r == nil || r.Pool == nil {
		return fmt.Errorf("database repository is not initialized")
	}

	// 🚀 構造化ログ：実行前のパラメータ記録
	slog.DebugContext(ctx, "Executing INSERT metadata", "file_name", f.FileName)

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
		// 🚀 エラーログ：何が原因で失敗したか属性を付けて記録
		slog.ErrorContext(ctx, "Database insert failed", "file_name", f.FileName, "error", err)
		return fmt.Errorf("failed to insert metadata: %w", err)
	}

	// ✨ 修正ポイント：成功ログを追加
	// 保存後の ID をログに出すことで、フロントエンドのログと DB の中身を即座に紐付けられます
	slog.InfoContext(ctx, "Metadata saved successfully",
		"db_id", f.ID,
		"file_name", f.FileName,
		"status", f.Status,
	)

	return nil
}

// 🚀 重要: インターフェースが "Create" という名前を求めている場合
// Create は SaveMetadata と同じ役割として実装します
func (r *Repository) Create(ctx context.Context, record *domain.FileMetadata) error {
	return r.SaveMetadata(ctx, record)
}

// 🚀 重要: インターフェースが UpdateStatus を求めている場合
// UpdateStatus はステータスを更新します（今回は実装を省略してもエラーは消えます）
func (r *Repository) UpdateStatus(ctx context.Context, id int64, status domain.TransferStatus) error {
	if r == nil || r.Pool == nil {
		return fmt.Errorf("database repository is not initialized")
	}
	query := `UPDATE file_metadata SET status = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2`
	_, err := r.Pool.Exec(ctx, query, status, id)

	if err != nil {
		// 🚀 ここで db_id (id) を出す！
		slog.ErrorContext(ctx, "Failed to update status", "db_id", id, "status", status, "error", err)
		return err
	}
	return err
}

// 🚀 重要: FindByID も戻り値の型まで一致させる
// FindByID はIDで検索します（今回は実装を省略してもエラーは消えます）
func (r *Repository) FindByID(ctx context.Context, id int64) (*domain.FileMetadata, error) {
	// 必要になったら実装しましょう。今は一旦 nil を返すだけでもコンパイルは通ります。
	return nil, fmt.Errorf("not implemented")
}

// FindAll は PostgreSQL からフィルタ条件に合致するメタデータ一覧を取得します
func (r *Repository) FindAll(ctx context.Context, q domain.FileSearchQuery) ([]*domain.FileMetadata, error) {
	if r == nil || r.Pool == nil {
		return nil, fmt.Errorf("database repository is not initialized")
	}

	// 1. ベースとなるクエリと、動的な引数を保持するスライスを用意
	// WHERE 1=1 は、後続の条件を "AND ..." で単純に繋げるための定石です。
	baseQuery := `
        SELECT id, file_name, file_size, status, source, tags, created_at, updated_at
        FROM file_metadata
        WHERE 1=1
    `
	args := []any{}
	argIdx := 1

	// 2. 🚀 動的にフィルタ条件を追加
	// タグが指定されている場合のみ、PostgreSQLの配列包含演算子 (@>) を追加
	if len(q.Tags) > 0 {
		baseQuery += fmt.Sprintf(" AND tags @> $%d", argIdx)
		args = append(args, q.Tags)
		argIdx++
	}

	// 3. 並び替えとページネーションの追加
	baseQuery += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, q.Limit, q.Offset)

	// ログに最終的なSQLを出力（デバッグ用）
	slog.DebugContext(ctx, "Executing filtered FindAll", "query", baseQuery, "tags", q.Tags)

	rows, err := r.Pool.Query(ctx, baseQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query metadata: %w", err)
	}
	defer rows.Close()

	var results []*domain.FileMetadata
	for rows.Next() {
		m := &domain.FileMetadata{}
		err := rows.Scan(
			&m.ID,
			&m.FileName,
			&m.FileSize,
			&m.Status,
			&m.Source,
			&m.Tags,
			&m.CreatedAt,
			&m.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		results = append(results, m)
	}

	return results, nil
}
