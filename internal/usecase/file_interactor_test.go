package usecase

import (
	"context"
	"file-transfer-api/internal/domain"
	"io"
	"testing"
)

// ベンチマーク用のモック（何もしないで成功を返す）
type benchMockRepo struct{}

// Saveの実装
func (m *benchMockRepo) Save(ctx context.Context, n string, r io.Reader) error { return nil }

// ✅ これを追加：本物の GCS リポジトリが Close を持っているので、
// インターフェースを満たすためにモックにも定義します。
func (m *benchMockRepo) Close() error {
	return nil
}

func BenchmarkUploadMultipleParallel(b *testing.B) { // *testing.B に修正
	// 1. 準備
	repo := &benchMockRepo{}
	// 第1引数: Storage用Repo, 第2引数: DB用Repo(今回はnil), 第3引数: Pipeline(今回はnil)
	interactor := NewFileInteractor(repo, nil, nil)

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
