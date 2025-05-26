package safety

import (
	"context"
	"sort"
	"sync"
	"time"
)

// InMemoryActionStore provides an in-memory implementation of ActionStore
type InMemoryActionStore struct {
	mu      sync.RWMutex
	records map[string][]ActionRecord // map[policyKey][]records
}

// NewInMemoryActionStore creates a new in-memory action store
func NewInMemoryActionStore() *InMemoryActionStore {
	return &InMemoryActionStore{
		records: make(map[string][]ActionRecord),
	}
}

// RecordAction stores an executed action
func (s *InMemoryActionStore) RecordAction(ctx context.Context, record ActionRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if record.ID == "" {
		record.ID = generateID()
	}

	s.records[record.PolicyKey] = append(s.records[record.PolicyKey], record)
	
	// Keep records sorted by timestamp
	sort.Slice(s.records[record.PolicyKey], func(i, j int) bool {
		return s.records[record.PolicyKey][i].Timestamp.After(s.records[record.PolicyKey][j].Timestamp)
	})

	return nil
}

// GetActionCount returns the number of actions for a policy in the time window
func (s *InMemoryActionStore) GetActionCount(ctx context.Context, policyKey string, since time.Time) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	records, exists := s.records[policyKey]
	if !exists {
		return 0, nil
	}

	count := 0
	for _, record := range records {
		if record.Timestamp.After(since) || record.Timestamp.Equal(since) {
			count++
		}
	}

	return count, nil
}

// GetRecentActions returns recent actions for a policy
func (s *InMemoryActionStore) GetRecentActions(ctx context.Context, policyKey string, limit int) ([]ActionRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	records, exists := s.records[policyKey]
	if !exists {
		return []ActionRecord{}, nil
	}

	// Records are already sorted by timestamp (newest first)
	if len(records) <= limit {
		result := make([]ActionRecord, len(records))
		copy(result, records)
		return result, nil
	}

	result := make([]ActionRecord, limit)
	copy(result, records[:limit])
	return result, nil
}

// GetLastAction returns the most recent action for a policy
func (s *InMemoryActionStore) GetLastAction(ctx context.Context, policyKey string) (*ActionRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	records, exists := s.records[policyKey]
	if !exists || len(records) == 0 {
		return nil, nil
	}

	// Records are sorted with newest first
	record := records[0]
	return &record, nil
}

// CleanupOldRecords removes records older than the retention period
func (s *InMemoryActionStore) CleanupOldRecords(ctx context.Context, before time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for policyKey, records := range s.records {
		filtered := make([]ActionRecord, 0)
		for _, record := range records {
			if record.Timestamp.After(before) {
				filtered = append(filtered, record)
			}
		}
		
		if len(filtered) == 0 {
			delete(s.records, policyKey)
		} else {
			s.records[policyKey] = filtered
		}
	}

	return nil
}

// generateID generates a unique ID for action records
func generateID() string {
	return time.Now().Format("20060102150405") + "-" + generateRandomString(8)
}

// generateRandomString generates a random string of specified length
func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}