package usecase

import (
	"context"
	"errors"
	"file-transfer-api/internal/domain"
	"io"
	"strings"
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

// モックの FindAll を新しいシグネチャに修正
func (m *benchMockMetaRepo) FindAll(ctx context.Context, q domain.FileSearchQuery) ([]*domain.FileMetadata, error) {
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

	testFiles := []*domain.File{
		domain.NewFile("success-1.txt", 10, nil),
		domain.NewFile(failFileName, 10, nil), // ここでエラーを発生させる
		domain.NewFile("success-2.txt", 10, nil),
	}

	// 2. 実行：context.Background() を渡す
	err := interactor.UploadMultipleParallel(context.Background(), testFiles)

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
