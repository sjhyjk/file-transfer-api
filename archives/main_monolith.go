// [Benchmark Baseline]
// クリーンアーキテクチャ採用前のモノリスな初期実装。
// 依存関係が密結合であり、並行処理への拡張性が低い状態を検証するために保存。

package main

import (
	"context"
	"fmt"
	"log"

	//"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"
)

func main() {
	ctx := context.Background()
	bucketName := "file-transfer-bucket-syou-20240121" // 画像と一致
	keyFile := "gcp-key.json"

	client, err := storage.NewClient(ctx, option.WithCredentialsFile(keyFile))
	if err != nil {
		log.Fatalf("GCPへの接続に失敗しました: %v", err)
	}
	defer client.Close()

	// テスト用のファイル名
	objectName := "connection-test.txt"

	// バケットに対して書き込みを開始する
	wc := client.Bucket(bucketName).Object(objectName).NewWriter(ctx)

	// 書き込む内容
	if _, err := wc.Write([]byte("Hello from WSL2 via Go!")); err != nil {
		log.Fatalf("ファイルの書き込みに失敗しました: %v", err)
	}

	// 完了（これをしないと保存されません）
	if err := wc.Close(); err != nil {
		log.Fatalf("クローズに失敗しました: %v", err)
	}

	fmt.Printf("✅ 通信成功！バケットに %s を作成しました。\n", objectName)
}
