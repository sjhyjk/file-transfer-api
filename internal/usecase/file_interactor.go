package usecase

import (
	"context"
	"file-transfer-api/internal/domain" // 自身のgo.modにあるモジュール名に合わせてください
	"fmt"
	"io"
	"log/slog"

	"golang.org/x/sync/errgroup"
	// 並行処理の同期に必要
)

// FileInteractor は、ファイル操作のビジネスロジックを管理します
type FileInteractor struct {
	repo         domain.FileRepository
	metadataRepo domain.MetadataRepository // ★ 追加：DB用
	pipeline     domain.DataPipeline       // ★ 追加：RAGなど後続処理への通知用
}

func NewFileInteractor(repo domain.FileRepository, metadataRepo domain.MetadataRepository, pipeline domain.DataPipeline) *FileInteractor {
	return &FileInteractor{
		repo:         repo,
		metadataRepo: metadataRepo, // ★ 注入
		pipeline:     pipeline,     // ★ 注入（Dependency Injection）
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

// UploadMultipleParallel は、Goroutine と errgroup を用いて複数のファイルを並行してアップロードします。
// Goの並行処理モデルを活かしたスループット最大化に加え、Fail-fast なエラー制御により
// 異常系におけるコンピューティングリソースの保護を両立しています。
func (i *FileInteractor) UploadMultipleParallel(ctx context.Context, files []*domain.File) error {
	// errgroup.WithContext により、一箇所でもエラーが出ると ctx がキャンセルされる
	eg, egCtx := errgroup.WithContext(ctx)

	for _, f := range files {
		f := f // ループ変数のキャプチャ
		// Go 1.22未満の場合は必要ですが、最新なら不要です

		eg.Go(func() error {
			slog.InfoContext(ctx, "🚀 [Parallel] アップロード開始", "file_name", f.Name)

			// 1. Storage（GCS）への保存
			if err := i.repo.Save(egCtx, f.Name, f.Content); err != nil {
				return fmt.Errorf("%s のアップロード失敗: %w", f.Name, err)
			}

			// 2. ★ DB（Cloud SQL）へのメタデータ保存 ★
			meta := &domain.FileMetadata{
				FileName: f.Name,
				FileSize: f.Size,
				Status:   domain.StatusCompleted, // アップロード成功したので完了とする
				Source:   "direct-upload",
				Tags:     []string{"parallel-upload", "test"},
			}

			if i.metadataRepo != nil {
				if err := i.metadataRepo.SaveMetadata(egCtx, meta); err != nil {
					// ★ ここでロールバック発動！
					// 失敗した時だけ GCS から消しに行く（補償トランザクション）
					// egCtx はキャンセルされている可能性があるため、Background を使うのが安全です
					// ロールバック処理にはキャンセルされていない context.Background() を使うのがコツです
					rollbackCtx := context.Background()
					_ = i.repo.Delete(rollbackCtx, f.Name)

					return fmt.Errorf("%s のメタデータ保存失敗: %w", f.Name, err)
				}
			}

			// 3. パイプライン通知
			// ★ 保存に成功したら、即座にパイプラインへ通知を開始する
			if i.pipeline != nil {
				if err := i.pipeline.NotifyNewFile(egCtx, f.Name); err != nil {
					return fmt.Errorf("%s の通知失敗: %w", f.Name, err)
				}
			}

			slog.InfoContext(ctx, "✅ [Parallel] 処理完了",
				"file_name", f.Name,
				"db_id", meta.ID,
				"scope", "GCS+DB+Notify",
			)
			return nil
		})
	}

	// eg.Wait() は最初のエラーを返し、その時点で他の全処理の ctx をキャンセルする
	if err := eg.Wait(); err != nil {
		slog.ErrorContext(ctx, "❌ 並行処理中にエラーが発生し、中断されました", "error", err)
		return err
	}

	return nil
}

// FetchMetadataList は、検索条件に基づいてファイルの一覧を取得します。
// タグによるフィルタリングに加え、DB負荷対策のバリデーションを適用しています。
func (i *FileInteractor) FetchMetadataList(ctx context.Context, tags []string, limit, offset int) ([]*domain.FileMetadata, error) {
	// 1. バリデーション（防衛的プログラミング）
	if limit > 100 {
		limit = 100
	}
	if limit <= 0 {
		limit = 10
	}

	// 2. 検索クエリの構築（Specification Pattern の適用）
	query := domain.FileSearchQuery{
		Tags:   tags,
		Limit:  limit,
		Offset: offset,
	}

	slog.InfoContext(ctx, "🔍 メタデータ検索を開始",
		"tags", query.Tags,
		"limit", query.Limit,
		"offset", query.Offset,
	)

	if i.metadataRepo == nil {
		return nil, fmt.Errorf("metadata repository is not initialized")
	}

	// 3. Repository の呼び出し（引数を構造体に変更）
	return i.metadataRepo.FindAll(ctx, query)
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
