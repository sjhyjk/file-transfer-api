package inmemory

import (
	"context"
	"file-transfer-api/internal/domain"
	"sync"
)

type InMemoryRepository struct {
	sync.RWMutex
	data   map[int64]*domain.FileMetadata
	nextID int64
}

// NewInMemoryRepository は初期化されたリポジトリを返します
func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{
		data: make(map[int64]*domain.FileMetadata),
	}
}

// Create は SaveMetadata を呼び出してインターフェースを満たします
func (r *InMemoryRepository) Create(ctx context.Context, f *domain.FileMetadata) error {
	r.Lock()
	defer r.Unlock()
	r.nextID++
	f.ID = r.nextID
	r.data[f.ID] = f
	return nil
}

func (r *InMemoryRepository) SaveMetadata(ctx context.Context, f *domain.FileMetadata) error {
	r.Lock()
	defer r.Unlock()
	r.nextID++
	f.ID = r.nextID
	r.data[f.ID] = f
	return nil
}

// インターフェースを満たすための他のメソッドも定義（中身は空でもOK）
func (r *InMemoryRepository) FindAll(ctx context.Context, q domain.FileSearchQuery) ([]*domain.FileMetadata, error) {
	return nil, nil
}

func (r *InMemoryRepository) UpdateStatus(ctx context.Context, id int64, status domain.TransferStatus) error {
	return nil
}

func (r *InMemoryRepository) FindByID(ctx context.Context, id int64) (*domain.FileMetadata, error) {
	r.RLock()
	defer r.RUnlock()
	return r.data[id], nil
}
