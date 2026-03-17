package usecase

import (
	"context"
	"file-transfer-api/internal/domain" // 自身のgo.modにあるモジュール名に合わせてください
	"fmt"
	"io"
	"sync" // 並行処理の同期に必要
)

// FileRepository は、保存先の具体的な実装を抽象化したインターフェースです
type FileRepository interface {
	Save(ctx context.Context, name string, data io.Reader) error
}

// FileInteractor は、ファイル操作のビジネスロジックを管理します
type FileInteractor struct {
	repo FileRepository
}

func NewFileInteractor(repo FileRepository) *FileInteractor {
	return &FileInteractor{repo: repo}
}

// UploadSingle は、単一のファイルをアップロードする手順を定義します
func (i *FileInteractor) UploadSingle(ctx context.Context, name string, size int64, content io.Reader) error {
	file := domain.NewFile(name, size, content)
	return i.repo.Save(ctx, file.Name, file.Content)
}

// UploadMultiple は、複数のファイルを並行してアップロードします
func (i *FileInteractor) UploadMultiple(ctx context.Context, files []*domain.File) error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(files)) // 各処理のエラーを回収するチャネル

	for _, f := range files {
		wg.Add(1) // 待ち行列に1つ追加

		// ゴルーチン（並行処理）の開始
		go func(file *domain.File) {
			defer wg.Done() // 終わったら待ち行列から1つ減らす

			fmt.Printf("🚀 アップロード開始: %s\n", file.Name)
			if err := i.repo.Save(ctx, file.Name, file.Content); err != nil {
				errChan <- fmt.Errorf("%s のアップロード失敗: %w", file.Name, err)
				return
			}
			fmt.Printf("✅ アップロード完了: %s\n", file.Name)
		}(f)
	}

	// すべてのゴルーチンが終わるのを待つ
	wg.Wait()
	close(errChan)

	// 1つでもエラーがあれば、最初のエラーを返す（簡易的な実装）
	for err := range errChan {
		if err != nil {
			return err
		}
	}

	return nil
}
