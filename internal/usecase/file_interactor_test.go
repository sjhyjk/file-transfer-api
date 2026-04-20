package usecase

import (
	"context"
	"file-transfer-api/internal/domain"
	"io"
	"testing"
)

// 1. Storage用モック
// ベンチマーク用のモック（何もしないで成功を返す）
type benchMockRepo struct{}

// Saveの実装
func (m *benchMockRepo) Save(ctx context.Context, n string, r io.Reader) error { return nil }

// ✅ これを追加：本物の GCS リポジトリが Close を持っているので、
// インターフェースを満たすためにモックにも定義します。
func (m *benchMockRepo) Close() error {
	return nil
}

// benchMockRepo に追加
func (m *benchMockRepo) Delete(ctx context.Context, n string) error {
	return nil // ベンチマーク用なので何もしない
}

// 2. DB（Metadata）用モック
type benchMockMetaRepo struct{}

func (m *benchMockMetaRepo) Create(ctx context.Context, r *domain.FileMetadata) error { return nil }
func (m *benchMockMetaRepo) SaveMetadata(ctx context.Context, r *domain.FileMetadata) error {
	return nil
}
func (m *benchMockMetaRepo) UpdateStatus(ctx context.Context, id int64, s domain.TransferStatus) error {
	return nil
}
func (m *benchMockMetaRepo) FindByID(ctx context.Context, id int64) (*domain.FileMetadata, error) {
	return nil, nil
}

// これを追加してインターフェースを満たす
func (m *benchMockMetaRepo) FindAll(ctx context.Context, l, o int) ([]*domain.FileMetadata, error) {
	return nil, nil
}

func BenchmarkUploadMultipleParallel(b *testing.B) { // *testing.B に修正
	// 1. 準備
	repo := &benchMockRepo{}
	metaRepo := &benchMockMetaRepo{} // nil ではなくモックを渡すように変更

	// 第1引数: Storage用Repo, 第2引数: DB用Repo, 第3引数: Pipeline(今回はnil)
	interactor := NewFileInteractor(repo, metaRepo, nil)

	// 10個のダミーファイルを生成
	files := make([]*domain.File, 10)
	for i := 0; i < 10; i++ {
		files[i] = &domain.File{Name: "bench.txt"}
	}

	b.ResetTimer() // 純粋なループ処理だけを計測するためにタイマーをリセット
	for i := 0; i < b.N; i++ {
		_ = interactor.UploadMultipleParallel(context.Background(), files)
	}
}
