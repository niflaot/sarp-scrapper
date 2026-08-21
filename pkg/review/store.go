package review

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"go.uber.org/zap"
)

var ErrListingNotFound = errors.New("listing was not found in the local cache")

const maxStateFileSize = 10 << 20

type Status string

const (
	StatusPending  Status = "pending"
	StatusReviewed Status = "reviewed"
	StatusIgnored  Status = "ignored"
)

type Item struct {
	Kind  string `json:"kind"`
	Key   string `json:"key"`
	Name  string `json:"name"`
	Value int64  `json:"value"`
	Icon  string `json:"icon,omitempty"`
}

type Record struct {
	Item
	Status    Status    `json:"status"`
	Available bool      `json:"available"`
	Notified  bool      `json:"notified"`
	FirstSeen time.Time `json:"firstSeen"`
	LastSeen  time.Time `json:"lastSeen"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type stateFile struct {
	Version int               `json:"version"`
	Records map[string]Record `json:"records"`
}

type Store struct {
	config Config
	log    *zap.Logger
	mu     sync.RWMutex
	state  stateFile
}

func NewStore(config Config, log *zap.Logger) (*Store, error) {
	store := &Store{config: config, log: log, state: stateFile{Version: 1, Records: make(map[string]Record)}}
	if err := store.load(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *Store) Reconcile(kind string, items []Item, now time.Time) ([]Record, []Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now = now.UTC()
	next := cloneState(s.state)
	for id, record := range next.Records {
		if record.Kind == kind && record.Available {
			record.Available = false
			next.Records[id] = record
		}
	}
	for _, item := range items {
		item.Kind = kind
		id := recordID(kind, item.Key)
		record, exists := next.Records[id]
		if !exists {
			record = Record{Item: item, Status: StatusPending, FirstSeen: now}
		}
		record.Item = item
		record.Available = true
		record.LastSeen = now
		record.UpdatedAt = now
		if record.Status == "" {
			record.Status = StatusPending
		}
		next.Records[id] = record
	}
	if err := s.persist(next); err != nil {
		return nil, nil, err
	}
	s.state = next
	visible, unnotified := recordsForKind(next, kind)
	return visible, unnotified, nil
}

func (s *Store) SetStatus(kind, key string, status Status, now time.Time) (Record, error) {
	if status != StatusReviewed && status != StatusIgnored {
		return Record{}, fmt.Errorf("unsupported listing status %q", status)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	next := cloneState(s.state)
	id := recordID(kind, key)
	record, found := next.Records[id]
	if !found {
		return Record{}, ErrListingNotFound
	}
	record.Status = status
	record.UpdatedAt = now.UTC()
	next.Records[id] = record
	if err := s.persist(next); err != nil {
		return Record{}, err
	}
	s.state = next
	return record, nil
}

func (s *Store) MarkNotified(kind, key string, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	next := cloneState(s.state)
	id := recordID(kind, key)
	record, found := next.Records[id]
	if !found {
		return ErrListingNotFound
	}
	record.Notified = true
	record.UpdatedAt = now.UTC()
	next.Records[id] = record
	if err := s.persist(next); err != nil {
		return err
	}
	s.state = next
	return nil
}

func (s *Store) Available() []Record {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]Record, 0)
	for _, record := range s.state.Records {
		if record.Available && record.Status != StatusIgnored {
			result = append(result, record)
		}
	}
	sortRecords(result)
	return result
}

func (s *Store) MinimumValue(kind string) int64 {
	switch kind {
	case "house":
		return s.config.HouseMinValue
	case "business":
		return s.config.BusinessMinValue
	case "warehouse":
		return s.config.WarehouseMinValue
	default:
		return 0
	}
}

func (s *Store) load() error {
	file, err := os.Open(s.config.StateFile)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("open listings state: %w", err)
	}
	defer file.Close()
	decoder := json.NewDecoder(io.LimitReader(file, maxStateFileSize+1))
	var loaded stateFile
	if err := decoder.Decode(&loaded); err != nil {
		return fmt.Errorf("decode listings state: %w", err)
	}
	if loaded.Version != 1 {
		return fmt.Errorf("unsupported listings state version %d", loaded.Version)
	}
	if loaded.Records == nil {
		loaded.Records = make(map[string]Record)
	}
	s.state = loaded
	s.log.Info("listings state loaded", zap.Int("records", len(loaded.Records)))
	return nil
}

func (s *Store) persist(state stateFile) error {
	directory := filepath.Dir(s.config.StateFile)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return fmt.Errorf("create listings state directory: %w", err)
	}
	temporary, err := os.CreateTemp(directory, ".listings-state-*")
	if err != nil {
		return fmt.Errorf("create temporary listings state: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0o600); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("secure temporary listings state: %w", err)
	}
	encoder := json.NewEncoder(temporary)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(state); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("encode listings state: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("sync listings state: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close listings state: %w", err)
	}
	if err := os.Rename(temporaryPath, s.config.StateFile); err != nil {
		return fmt.Errorf("replace listings state: %w", err)
	}
	return nil
}

func recordsForKind(state stateFile, kind string) ([]Record, []Record) {
	visible := make([]Record, 0)
	unnotified := make([]Record, 0)
	for _, record := range state.Records {
		if record.Kind != kind || !record.Available {
			continue
		}
		if record.Status != StatusIgnored {
			visible = append(visible, record)
		}
		if record.Status != StatusIgnored && !record.Notified {
			unnotified = append(unnotified, record)
		}
	}
	sortRecords(visible)
	sortRecords(unnotified)
	return visible, unnotified
}

func sortRecords(records []Record) {
	sort.Slice(records, func(i, j int) bool {
		if records[i].Kind != records[j].Kind {
			return records[i].Kind < records[j].Kind
		}
		if records[i].Name != records[j].Name {
			return records[i].Name < records[j].Name
		}
		return records[i].Key < records[j].Key
	})
}

func cloneState(source stateFile) stateFile {
	result := stateFile{Version: source.Version, Records: make(map[string]Record, len(source.Records))}
	for key, value := range source.Records {
		result.Records[key] = value
	}
	return result
}

func recordID(kind, key string) string { return kind + "/" + url.PathEscape(key) }
