package safety

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInMemoryActionStore_ConcurrentAccess(t *testing.T) {
	store := NewInMemoryActionStore()
	ctx := context.Background()

	// Test concurrent writes
	var wg sync.WaitGroup
	numGoroutines := 10
	recordsPerGoroutine := 100

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()
			
			for j := 0; j < recordsPerGoroutine; j++ {
				record := ActionRecord{
					PolicyKey:  "default/test-policy",
					ActionName: "test-action",
					ActionType: "restart",
					TargetKey:  "Pod/default/test-pod",
					Success:    true,
					Timestamp:  time.Now(),
				}
				
				err := store.RecordAction(ctx, record)
				assert.NoError(t, err)
			}
		}(i)
	}

	wg.Wait()

	// Verify all records were stored
	count, err := store.GetActionCount(ctx, "default/test-policy", time.Now().Add(-1*time.Hour))
	require.NoError(t, err)
	assert.Equal(t, numGoroutines*recordsPerGoroutine, count)
}

func TestInMemoryActionStore_EdgeCases(t *testing.T) {
	store := NewInMemoryActionStore()
	ctx := context.Background()

	// Test empty policy key
	count, err := store.GetActionCount(ctx, "nonexistent/policy", time.Now().Add(-1*time.Hour))
	require.NoError(t, err)
	assert.Equal(t, 0, count)

	// Test GetLastAction on empty store
	last, err := store.GetLastAction(ctx, "nonexistent/policy")
	require.NoError(t, err)
	assert.Nil(t, last)

	// Test GetRecentActions on empty store
	recent, err := store.GetRecentActions(ctx, "nonexistent/policy", 10)
	require.NoError(t, err)
	assert.Empty(t, recent)

	// Test cleanup on empty store
	err = store.CleanupOldRecords(ctx, time.Now())
	require.NoError(t, err)
}

func TestInMemoryActionStore_Sorting(t *testing.T) {
	store := NewInMemoryActionStore()
	ctx := context.Background()

	// Add records in random order
	timestamps := []time.Time{
		time.Now().Add(-5 * time.Minute),
		time.Now().Add(-1 * time.Hour),
		time.Now().Add(-30 * time.Minute),
		time.Now().Add(-2 * time.Minute),
		time.Now().Add(-45 * time.Minute),
	}

	for i, ts := range timestamps {
		record := ActionRecord{
			PolicyKey:  "default/test-policy",
			ActionName: "action" + string(rune('A'+i)),
			Timestamp:  ts,
		}
		err := store.RecordAction(ctx, record)
		require.NoError(t, err)
	}

	// Get recent actions and verify they're sorted newest first
	recent, err := store.GetRecentActions(ctx, "default/test-policy", 5)
	require.NoError(t, err)
	require.Len(t, recent, 5)

	// Check ordering (newest first)
	assert.Equal(t, "actionD", recent[0].ActionName) // 2 minutes ago
	assert.Equal(t, "actionA", recent[1].ActionName) // 5 minutes ago
	assert.Equal(t, "actionC", recent[2].ActionName) // 30 minutes ago
	assert.Equal(t, "actionE", recent[3].ActionName) // 45 minutes ago
	assert.Equal(t, "actionB", recent[4].ActionName) // 1 hour ago

	// Verify timestamps are in descending order
	for i := 1; i < len(recent); i++ {
		assert.True(t, recent[i-1].Timestamp.After(recent[i].Timestamp) || 
			recent[i-1].Timestamp.Equal(recent[i].Timestamp))
	}
}

func TestInMemoryActionStore_CleanupOldRecords(t *testing.T) {
	store := NewInMemoryActionStore()
	ctx := context.Background()

	now := time.Now()
	
	// Add records with different ages
	records := []struct {
		name string
		age  time.Duration
	}{
		{"very-old", 48 * time.Hour},
		{"old", 25 * time.Hour},
		{"recent", 23 * time.Hour},
		{"new", 1 * time.Hour},
		{"newest", 10 * time.Minute},
	}

	for _, r := range records {
		record := ActionRecord{
			PolicyKey:  "default/test-policy",
			ActionName: r.name,
			Timestamp:  now.Add(-r.age),
		}
		err := store.RecordAction(ctx, record)
		require.NoError(t, err)
	}

	// Clean up records older than 24 hours
	err := store.CleanupOldRecords(ctx, now.Add(-24*time.Hour))
	require.NoError(t, err)

	// Check remaining records
	recent, err := store.GetRecentActions(ctx, "default/test-policy", 10)
	require.NoError(t, err)
	require.Len(t, recent, 3)

	// Verify only recent records remain
	names := make([]string, len(recent))
	for i, r := range recent {
		names[i] = r.ActionName
	}
	assert.Contains(t, names, "newest")
	assert.Contains(t, names, "new")
	assert.Contains(t, names, "recent")
	assert.NotContains(t, names, "old")
	assert.NotContains(t, names, "very-old")
}

func TestGenerateID(t *testing.T) {
	// Test ID generation
	id1 := generateID()
	id2 := generateID()

	// IDs should be unique
	assert.NotEqual(t, id1, id2)

	// IDs should have expected format
	assert.Regexp(t, `^\d{14}-[a-z0-9]{8}$`, id1)
	assert.Regexp(t, `^\d{14}-[a-z0-9]{8}$`, id2)
}

func TestInMemoryActionStore_MultiplePolices(t *testing.T) {
	store := NewInMemoryActionStore()
	ctx := context.Background()

	// Add records for multiple policies
	policies := []string{
		"default/policy1",
		"default/policy2",
		"namespace1/policy3",
	}

	for _, policy := range policies {
		for i := 0; i < 5; i++ {
			record := ActionRecord{
				PolicyKey:  policy,
				ActionName: "action",
				Timestamp:  time.Now().Add(-time.Duration(i) * time.Minute),
			}
			err := store.RecordAction(ctx, record)
			require.NoError(t, err)
		}
	}

	// Verify each policy has its own records
	for _, policy := range policies {
		count, err := store.GetActionCount(ctx, policy, time.Now().Add(-1*time.Hour))
		require.NoError(t, err)
		assert.Equal(t, 5, count)
	}

	// Clean up one policy's records
	// First, get all records for policy1
	store.mu.Lock()
	store.records["default/policy1"] = []ActionRecord{}
	store.mu.Unlock()

	// Verify only policy1 was affected
	count, err := store.GetActionCount(ctx, "default/policy1", time.Now().Add(-1*time.Hour))
	require.NoError(t, err)
	assert.Equal(t, 0, count)

	count, err = store.GetActionCount(ctx, "default/policy2", time.Now().Add(-1*time.Hour))
	require.NoError(t, err)
	assert.Equal(t, 5, count)
}