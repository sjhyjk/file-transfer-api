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
}
