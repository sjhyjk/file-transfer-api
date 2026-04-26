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

	// deferの中でエラーをチェックし、関数の戻り値 err に代入する
	defer func() {
		closeErr := f.Close()
		if err == nil { // 本体の処理が成功している場合のみ、Closeのエラーを反映
			err = closeErr
		}
	}()

	_, err = io.Copy(f, data)
	if err != nil {
		return fmt.Errorf("failed to copy data: %w", err)
	}

	return nil
}

func (r *LocalRepository) Delete(ctx context.Context, name string) error {
	return os.Remove(filepath.Join(r.baseDir, name))
}
