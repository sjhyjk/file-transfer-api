// internal/infra/repository/sql/fs.go
package sql

import "embed"

// 🚀 このファイルから見て「../../../../migrations」を埋め込む
// go:embed というマジックコメントは相対パスで外側のフォルダを追えます
// ※ Go 1.22 以降は親ディレクトリの embed も可能になっています。

//go:embed all:../../../../migrations/*.sql
var migrationFS embed.FS
