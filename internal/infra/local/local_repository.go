package local

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type LocalRepository struct {
	baseDir string
}

func NewLocalRepository(baseDir string) *LocalRepository {
	// 保存用ディレクトリがなければ作る
	_ = os.MkdirAll(baseDir, 0755)
	return &LocalRepository{baseDir: baseDir}
}

// Close はインターフェースを満たすために定義します（ローカル保存では何もしません）
func (r *LocalRepository) Close() error {
	return nil
}

func (r *LocalRepository) Save(ctx context.Context, name string, data io.Reader) error {
	path := filepath.Join(r.baseDir, name)
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("local save failed: %w", err)
	}
	defer f.Close()

	_, err = io.Copy(f, data)
	return err
}

func (r *LocalRepository) Delete(ctx context.Context, name string) error {
	return os.Remove(filepath.Join(r.baseDir, name))
}
