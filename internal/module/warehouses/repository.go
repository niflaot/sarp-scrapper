package warehouses

import (
	"sync"
	"time"
)

type Repository interface {
	Snapshot() Snapshot
	RecordAttempt(time.Time)
	Replace([]Warehouse, time.Time)
	RecordFailure(error)
	SetNextRun(*time.Time)
}

type MemoryRepository struct {
	mu       sync.RWMutex
	snapshot Snapshot
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{snapshot: Snapshot{Warehouses: []Warehouse{}}}
}

func (r *MemoryRepository) Snapshot() Snapshot {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := r.snapshot
	result.Warehouses = append([]Warehouse(nil), r.snapshot.Warehouses...)
	return result
}

func (r *MemoryRepository) RecordAttempt(at time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.snapshot.LastAttempt = timePointer(at)
}

func (r *MemoryRepository) Replace(warehouses []Warehouse, at time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.snapshot.Warehouses = append([]Warehouse(nil), warehouses...)
	r.snapshot.Count = len(warehouses)
	r.snapshot.LastSuccess = timePointer(at)
	r.snapshot.LastError = ""
}

func (r *MemoryRepository) RecordFailure(err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.snapshot.LastError = err.Error()
}

func (r *MemoryRepository) SetNextRun(at *time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if at == nil {
		r.snapshot.NextRun = nil
		return
	}
	r.snapshot.NextRun = timePointer(*at)
}

func timePointer(value time.Time) *time.Time { copy := value; return &copy }
