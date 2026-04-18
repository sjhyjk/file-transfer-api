// [プロジェクトルート]/assets.go
package file_transfer_api // go.mod のモジュール名と一致させる

import "embed"

// 🚀 プロジェクトルートにある migrations フォルダを直接指定
//
//go:embed migrations/*.sql
var MigrationFS embed.FS
