// internal/usecase/file_interactor_test.go

package usecase

import (
	"context"
	"errors"
	"file-transfer-api/internal/domain"
	"fmt"
	"io"
	"strings"
	"testing"
)

// 1. Storage用モック
// ベンチマーク用のモック（何もしないで成功を返す）
type benchMockRepo struct{}

// Saveの実装
func (m *benchMockRepo) Save(ctx context.Context, n string, r io.Reader) error { return nil }

// 本物の GCS リポジトリが Close を持っているので、
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

func (m *benchMockMetaRepo) Create(ctx context.Context, r *domain.FileMetadata) error {
	r.ID = 123 // モックとしてIDを割り振るシミュレーション
	return nil
}
func (m *benchMockMetaRepo) SaveMetadata(ctx context.Context, r *domain.FileMetadata) error {
	return nil
}
func (m *benchMockMetaRepo) UpdateStatus(ctx context.Context, id int64, s domain.TransferStatus) error {
	return nil
}
func (m *benchMockMetaRepo) FindByID(ctx context.Context, id int64) (*domain.FileMetadata, error) {
	return &domain.FileMetadata{ID: id, Status: domain.StatusCompleted}, nil
}

// モックの FindAll を新しいシグネチャに修正
func (m *benchMockMetaRepo) FindAll(ctx context.Context, q domain.FileSearchQuery) ([]*domain.FileMetadata, error) {
	return []*domain.FileMetadata{}, nil
}

func BenchmarkUploadMultipleParallel(b *testing.B) { // *testing.B に修正
	// 1. 準備
	repo := &benchMockRepo{}
	metaRepo := &benchMockMetaRepo{} // nil ではなくモックを渡すように変更

	// 第1引数: Storage用Repo, 第2引数: DB用Repo, 第3引数: Pipeline(今回はnil)
	interactor := NewFileInteractor(repo, metaRepo, nil)

	ctx := context.Background()

	b.ResetTimer() // 純粋なループ処理だけを計測するためにタイマーをリセット
	for i := 0; i < 10; i++ {
		// ⚠️ io.Reader は一度読むと消費されてしまうため、ループ毎にフレッシュな状態にする
		b.StopTimer()
		// 10個のダミーファイルを生成
		files := make([]*domain.File, 10)
		for j := 0; j < 10; j++ {
			// ✅ NewFile を使うか、エラーを処理する
			f, _ := domain.NewFile(fmt.Sprintf("bench-%d.txt", j), 100, strings.NewReader("dummy"))
			files[j] = f
		}
		b.StartTimer()

		_ = interactor.UploadMultipleParallel(ctx, "bench-tenant", files)
	}
}

// 3. テスト専用：エラーを発生させるためのストレージモック
type errorMockRepo struct {
	benchMockRepo        // 既存の成功用モックを埋め込み（DeleteやCloseを使い回す）
	failOnName    string // この名前のファイルが来たらエラーにする
}

// Save をオーバーライドして、特定の条件で失敗させる
func (m *errorMockRepo) Save(ctx context.Context, n string, r io.Reader) error {
	if n == m.failOnName {
		return errors.New("simulated storage error")
	}
	return nil
}

// --- [テストコードの追加] ---

func TestUploadMultipleParallel_FailFast(t *testing.T) {
	// 1. 準備：2番目のファイルだけ失敗するように設定
	failFileName := "fail-me.txt"
	repo := &errorMockRepo{failOnName: failFileName}
	metaRepo := &benchMockMetaRepo{}

	interactor := NewFileInteractor(repo, metaRepo, nil)

	f1, _ := domain.NewFile("success-1.txt", 10, strings.NewReader("dummy"))
	f2, _ := domain.NewFile(failFileName, 10, strings.NewReader("dummy")) // ここでエラーを発生させる
	f3, _ := domain.NewFile("success-2.txt", 10, strings.NewReader("dummy"))

	testFiles := []*domain.File{f1, f2, f3}

	// 2. 実行：context.Background() を渡す
	err := interactor.UploadMultipleParallel(context.Background(), "test-tenant", testFiles)

	// 3. 検証：errgroup によってエラーが呼び出し元に返ってくるか
	if err == nil {
		t.Fatal("エラーが発生するはずですが、nilが返されました")
	}

	// 文字列の完全一致ではなく、中身が含まれているかを確認する
	expectedPart := "simulated storage error"
	if !strings.Contains(err.Error(), expectedPart) {
		t.Errorf("エラーメッセージに '%s' が含まれていません: %v", expectedPart, err)
	}

	t.Logf("✅ 期待通りエラーをキャッチし、並行処理を中断しました: %v", err)
}
