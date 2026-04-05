package usecase

import (
	"context"
	"file-transfer-api/internal/domain" // 自身のgo.modにあるモジュール名に合わせてください
	"fmt"
	"io"
	"sync" // 並行処理の同期に必要
)

// FileInteractor は、ファイル操作のビジネスロジックを管理します
type FileInteractor struct {
	repo     domain.FileRepository
	pipeline domain.DataPipeline // ★ 追加：RAGなど後続処理への通知用
}

func NewFileInteractor(repo domain.FileRepository, pipeline domain.DataPipeline) *FileInteractor {
	return &FileInteractor{
		repo:     repo,
		pipeline: pipeline, // ★ 注入（Dependency Injection）
	}
}

// UploadSingle は、単一のファイルをアップロードする手順を定義します
func (i *FileInteractor) UploadSingle(ctx context.Context, name string, size int64, content io.Reader) error {
	file := domain.NewFile(name, size, content)
	if err := i.repo.Save(ctx, file.Name, file.Content); err != nil {
		return err
	}

	// ★ 保存成功後、パイプラインに通知
	if i.pipeline != nil {
		return i.pipeline.NotifyNewFile(ctx, file.Name)
	}
	return nil
}

// UploadMultipleParallel は、Goroutine を用いて複数のファイルを並行してアップロードします。
// Goの並行処理モデル（Concurrency）を活かし、I/O待ち時間を最小化する「進化的アーキテクチャ」の主実装です。
func (i *FileInteractor) UploadMultipleParallel(ctx context.Context, files []*domain.File) error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(files)) // チャネルによるエラー集約

	for _, f := range files {
		wg.Add(1)

		// 各ファイルのアップロードを非同期（Goroutine）で実行
		go func(file *domain.File) {
			defer wg.Done()

			fmt.Printf("🚀 [Parallel] アップロード開始: %s\n", file.Name)
			if err := i.repo.Save(ctx, file.Name, file.Content); err != nil {
				// エラーが発生した場合はチャネル経由で呼び出し元に通知
				errChan <- fmt.Errorf("%s のアップロード失敗: %w", file.Name, err)
				return
			}

			// ★ 保存に成功したら、即座にパイプラインへ通知を開始する
			if i.pipeline != nil {
				if err := i.pipeline.NotifyNewFile(ctx, file.Name); err != nil {
					errChan <- fmt.Errorf("%s の通知失敗: %w", file.Name, err)
					return
				}
			}

			fmt.Printf("✅ [Parallel] アップロード完了 & 通知済: %s\n", file.Name)
		}(f)
	}

	// 全ての Goroutine が完了するのを待機
	wg.Wait()
	close(errChan)

	// 最初に見つかったエラーを返す（検証用のため簡易実装）
	for err := range errChan {
		if err != nil {
			return err
		}
	}
	return nil
}

// UploadMultipleSerial は、複数のファイルを1つずつ順番にアップロードします。
// 【検証用】並行処理（Parallel）の優位性を定量的に実証するための比較用パスとして実装しています。
func (i *FileInteractor) UploadMultipleSerial(ctx context.Context, files []*domain.File) error {
	for _, f := range files {
		fmt.Printf("⏳ [Serial] アップロード開始: %s\n", f.Name)
		// 1つのアップロードが完了するまで次のループへ進まない（逐次処理）
		if err := i.repo.Save(ctx, f.Name, f.Content); err != nil {
			return fmt.Errorf("%s のアップロード失敗: %w", f.Name, err)
		}
		fmt.Printf("✅ [Serial] アップロード完了: %s\n", f.Name)
	}
	return nil
}
