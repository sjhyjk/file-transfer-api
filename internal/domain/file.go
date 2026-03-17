package domain

import "io"

// File はこのアプリケーションで扱う「ファイル」の定義です
type File struct {
	Name    string
	Size    int64
	Content io.Reader // メモリ効率を考え、[]byteではなくストリーム(io.Reader)で扱えるようにします
}

// NewFile はドメインモデルの生成を管理します（バリデーションなどを後で追加できます）
func NewFile(name string, size int64, content io.Reader) *File {
	return &File{
		Name:    name,
		Size:    size,
		Content: content,
	}
}
