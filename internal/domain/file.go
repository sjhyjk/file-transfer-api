// internal/domain/file.go

package domain

import (
	"fmt"
	"io"
)

// File はこのアプリケーションで扱う「ファイル」の定義です
type File struct {
	Name    string
	Size    int64
	Content io.Reader // メモリ効率を考え、[]byteではなくストリーム(io.Reader)で扱えるようにします
}

// NewFile はドメインモデルの生成を管理し、不整合なデータが作られるのを防ぎます
func NewFile(name string, size int64, content io.Reader) (*File, error) {
	if name == "" {
		return nil, fmt.Errorf("file name cannot be empty")
	}
	if size < 0 {
		return nil, fmt.Errorf("invalid file size")
	}

	return &File{
		Name:    name,
		Size:    size,
		Content: content,
	}, nil
}

// FileMetadata を生成するビジネスルールをドメインモデルに持たせる
func (f *File) ToMetadata(source string, tags ...string) *FileMetadata {
	if len(tags) == 0 {
		tags = []string{"auto-generated"} // デフォルトタグなど
	}
	return &FileMetadata{
		FileName: f.Name,
		FileSize: f.Size,
		Status:   StatusPending,
		Source:   source,
		Tags:     tags,
	}
}
