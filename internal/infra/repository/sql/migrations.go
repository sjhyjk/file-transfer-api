package sql

import (
	"embed"
	"errors"
	"fmt"
	"log/slog"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/golang-migrate/migrate/v4/source/iofs" // 🚀 これが必要
)

// 🚀 [重要] migrations フォルダをこのパッケージの直下に移動するか、
//    それが嫌な場合は、この階層に migrations/ へのシンボリックリンクを置くか、
//    あるいは「ルート」で embed したものを main から渡す形に統一します。

// 最も確実なのは、main.go があるディレクトリか、その配下で embed することです。
// 今回は「インフラ層で完結」させるため、fs.go を使わずにここに直接書きます。

// RunMigrations は指定されたDB URLに対してマイグレーションを実行します
// RunMigrations は外部から資産 (fs) を受け取る設計にする (DI)
func RunMigrations(dbURL string, fs embed.FS) error {
	// 🚀 iofs (In-Memory File System) ドライバを作成
	d, err := iofs.New(fs, "migrations") // fs 内の "migrations" フォルダを参照
	if err != nil {
		return fmt.Errorf("iofs driver creation failed: %w", err)
	}

	// 🚀 iofs をソースとして指定
	m, err := migrate.NewWithSourceInstance("iofs", d, dbURL)
	if err != nil {
		return fmt.Errorf("migration instance creation failed: %w", err)
	}

	// 最新の状態までアップデート
	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			slog.Info("Database is already up-to-date (no changes)")
			return nil
		}
		return fmt.Errorf("migration up failed: %w", err)
	}

	slog.Info("🎉 Database migrations completed successfully")
	return nil
}
