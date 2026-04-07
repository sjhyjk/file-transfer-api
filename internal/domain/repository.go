package domain

import (
	"context"
	"io"
)

// FileRepository は、保存先の具体的な実装を抽象化したインターフェースです
// FileRepository は外部ストレージ操作の抽象化インターフェース
// domain層に置くことで、全ての層から参照可能になります
type FileRepository interface {
	Save(ctx context.Context, name string, data io.Reader) error
	Close() error // これで main.go の defer が動くようになる

	// 今後の深化：ビジネスルールに基づくバッチ処理やリトライの抽象化
	// FindAllByStatus(ctx context.Context, status TransferStatus) ([]*File, error)
}

// DataPipeline は、保存されたデータを後続の処理（RAGのインジェストなど）へ
// 渡すための抽象的な「出口」を定義します。
type DataPipeline interface {
	NotifyNewFile(ctx context.Context, fileName string) error
}
