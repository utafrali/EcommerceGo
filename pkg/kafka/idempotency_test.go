package kafka

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

// ---------------------------------------------------------------------------
// MemoryIdempotencyStore tests
// ---------------------------------------------------------------------------

func TestMemoryIdempotencyStore_AddAndContains(t *testing.T) {
	store := NewMemoryIdempotencyStore(1 * time.Minute)
	ctx := context.Background()

	if err := store.Add(ctx, "evt-1"); err != nil {
		t.Fatalf("Add() returned error: %v", err)
	}

	got, err := store.Contains(ctx, "evt-1")
	if err != nil {
		t.Fatalf("Contains() returned error: %v", err)
	}
	if !got {
		t.Error("Contains(evt-1) = false, want true after Add")
	}
}

func TestMemoryIdempotencyStore_ContainsUnknown(t *testing.T) {
	store := NewMemoryIdempotencyStore(1 * time.Minute)
	ctx := context.Background()

	got, err := store.Contains(ctx, "unknown-id")
	if err != nil {
		t.Fatalf("Contains() returned error: %v", err)
	}
	if got {
		t.Error("Contains(unknown-id) = true, want false for unknown ID")
	}
}

func TestMemoryIdempotencyStore_Expiry(t *testing.T) {
	store := NewMemoryIdempotencyStore(10 * time.Millisecond)
	ctx := context.Background()

	if err := store.Add(ctx, "evt-expire"); err != nil {
		t.Fatalf("Add() returned error: %v", err)
	}

	// Immediately after add, should exist.
	got, err := store.Contains(ctx, "evt-expire")
	if err != nil {
		t.Fatalf("Contains() returned error: %v", err)
	}
	if !got {
		t.Error("Contains(evt-expire) = false immediately after Add, want true")
	}

	// Wait for TTL to expire.
	time.Sleep(20 * time.Millisecond)

	got, err = store.Contains(ctx, "evt-expire")
	if err != nil {
		t.Fatalf("Contains() returned error: %v", err)
	}
	if got {
		t.Error("Contains(evt-expire) = true after TTL expiry, want false")
	}
}

func TestMemoryIdempotencyStore_Len(t *testing.T) {
	store := NewMemoryIdempotencyStore(1 * time.Minute)
	ctx := context.Background()

	if store.Len() != 0 {
		t.Errorf("Len() = %d for new store, want 0", store.Len())
	}

	_ = store.Add(ctx, "a")
	_ = store.Add(ctx, "b")
	_ = store.Add(ctx, "c")

	if store.Len() != 3 {
		t.Errorf("Len() = %d after 3 adds, want 3", store.Len())
	}
}

func TestMemoryIdempotencyStore_ConcurrentAccess(t *testing.T) {
	store := NewMemoryIdempotencyStore(1 * time.Minute)
	ctx := context.Background()

	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			eventID := "evt-concurrent"
			_ = store.Add(ctx, eventID)
			_, _ = store.Contains(ctx, eventID)
		}(i)
	}

	wg.Wait()

	// The store should have exactly 1 entry (all goroutines wrote the same key).
	if store.Len() != 1 {
		t.Errorf("Len() = %d after concurrent adds of same key, want 1", store.Len())
	}

	got, err := store.Contains(ctx, "evt-concurrent")
	if err != nil {
		t.Fatalf("Contains() returned error: %v", err)
	}
	if !got {
		t.Error("Contains(evt-concurrent) = false after concurrent adds, want true")
	}
}

func TestMemoryIdempotencyStore_MultipleAdds(t *testing.T) {
	store := NewMemoryIdempotencyStore(1 * time.Minute)
	ctx := context.Background()

	// Adding the same ID multiple times should be idempotent.
	for i := 0; i < 5; i++ {
		if err := store.Add(ctx, "evt-dup"); err != nil {
			t.Fatalf("Add() iteration %d returned error: %v", i, err)
		}
	}

	if store.Len() != 1 {
		t.Errorf("Len() = %d after adding same ID 5 times, want 1", store.Len())
	}

	got, err := store.Contains(ctx, "evt-dup")
	if err != nil {
		t.Fatalf("Contains() returned error: %v", err)
	}
	if !got {
		t.Error("Contains(evt-dup) = false after multiple adds, want true")
	}
}

// ---------------------------------------------------------------------------
// IdempotentHandler tests
// ---------------------------------------------------------------------------

// testEvent creates an Event with the given event ID. We construct it directly
// rather than using NewEvent which calls uuid.New().
func testEvent(eventID string) *Event {
	return &Event{
		EventID:     eventID,
		EventType:   "test.event",
		AggregateID: "agg-123",
	}
}

func TestIdempotentHandler_FirstCall_ProcessesMessage(t *testing.T) {
	store := NewMemoryIdempotencyStore(1 * time.Minute)
	logger := testLogger()

	var callCount int32
	inner := func(ctx context.Context, event *Event) error {
		atomic.AddInt32(&callCount, 1)
		return nil
	}

	handler := IdempotentHandler(store, inner, logger)

	event := testEvent("evt-first")
	if err := handler(context.Background(), event); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if atomic.LoadInt32(&callCount) != 1 {
		t.Errorf("inner handler called %d times, want 1", atomic.LoadInt32(&callCount))
	}
}

func TestIdempotentHandler_DuplicateCall_SkipsMessage(t *testing.T) {
	store := NewMemoryIdempotencyStore(1 * time.Minute)
	logger := testLogger()

	var callCount int32
	inner := func(ctx context.Context, event *Event) error {
		atomic.AddInt32(&callCount, 1)
		return nil
	}

	handler := IdempotentHandler(store, inner, logger)
	event := testEvent("evt-dup")

	// First call: should process.
	if err := handler(context.Background(), event); err != nil {
		t.Fatalf("first call returned error: %v", err)
	}

	// Second call: should skip (duplicate).
	if err := handler(context.Background(), event); err != nil {
		t.Fatalf("second call returned error: %v", err)
	}

	if atomic.LoadInt32(&callCount) != 1 {
		t.Errorf("inner handler called %d times, want 1 (second call should be skipped)", atomic.LoadInt32(&callCount))
	}
}

func TestIdempotentHandler_EmptyEventID_PassesThrough(t *testing.T) {
	store := NewMemoryIdempotencyStore(1 * time.Minute)
	logger := testLogger()

	var callCount int32
	inner := func(ctx context.Context, event *Event) error {
		atomic.AddInt32(&callCount, 1)
		return nil
	}

	handler := IdempotentHandler(store, inner, logger)

	// Event with empty EventID -- cannot deduplicate, should always pass through.
	event := testEvent("")
	for i := 0; i < 3; i++ {
		if err := handler(context.Background(), event); err != nil {
			t.Fatalf("call %d returned error: %v", i+1, err)
		}
	}

	if atomic.LoadInt32(&callCount) != 3 {
		t.Errorf("inner handler called %d times, want 3 (empty EventID should always pass through)", atomic.LoadInt32(&callCount))
	}
}

func TestIdempotentHandler_HandlerError_DoesNotMarkProcessed(t *testing.T) {
	store := NewMemoryIdempotencyStore(1 * time.Minute)
	logger := testLogger()

	handlerErr := errors.New("processing failed")
	var callCount int32
	inner := func(ctx context.Context, event *Event) error {
		atomic.AddInt32(&callCount, 1)
		return handlerErr
	}

	handler := IdempotentHandler(store, inner, logger)
	event := testEvent("evt-err")

	// First call: handler returns error, event should NOT be marked as processed.
	err := handler(context.Background(), event)
	if !errors.Is(err, handlerErr) {
		t.Fatalf("expected handlerErr, got: %v", err)
	}

	// Verify the event ID was NOT stored.
	exists, storeErr := store.Contains(context.Background(), "evt-err")
	if storeErr != nil {
		t.Fatalf("store.Contains() returned error: %v", storeErr)
	}
	if exists {
		t.Error("event ID was stored despite handler error, want not stored")
	}

	// Second call: should still be processed (not skipped) since first call failed.
	err = handler(context.Background(), event)
	if !errors.Is(err, handlerErr) {
		t.Fatalf("expected handlerErr on retry, got: %v", err)
	}

	if atomic.LoadInt32(&callCount) != 2 {
		t.Errorf("inner handler called %d times, want 2 (both calls should process since first failed)", atomic.LoadInt32(&callCount))
	}
}

func TestIdempotentHandler_StoreError_ProcessesAnyway(t *testing.T) {
	logger := testLogger()

	// Use a failing store that always returns errors.
	failStore := &failingIdempotencyStore{}

	var callCount int32
	inner := func(ctx context.Context, event *Event) error {
		atomic.AddInt32(&callCount, 1)
		return nil
	}

	handler := IdempotentHandler(failStore, inner, logger)
	event := testEvent("evt-store-fail")

	// Even though store.Contains fails, handler should still be called (fail-open).
	if err := handler(context.Background(), event); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if atomic.LoadInt32(&callCount) != 1 {
		t.Errorf("inner handler called %d times, want 1 (fail-open should still process)", atomic.LoadInt32(&callCount))
	}
}

func TestIdempotentHandler_DifferentEventIDs_BothProcessed(t *testing.T) {
	store := NewMemoryIdempotencyStore(1 * time.Minute)
	logger := testLogger()

	var callCount int32
	inner := func(ctx context.Context, event *Event) error {
		atomic.AddInt32(&callCount, 1)
		return nil
	}

	handler := IdempotentHandler(store, inner, logger)

	event1 := testEvent("evt-aaa")
	event2 := testEvent("evt-bbb")

	if err := handler(context.Background(), event1); err != nil {
		t.Fatalf("handler(event1) returned error: %v", err)
	}
	if err := handler(context.Background(), event2); err != nil {
		t.Fatalf("handler(event2) returned error: %v", err)
	}

	if atomic.LoadInt32(&callCount) != 2 {
		t.Errorf("inner handler called %d times, want 2 (different event IDs should both be processed)", atomic.LoadInt32(&callCount))
	}

	// Verify both are now in the store.
	for _, id := range []string{"evt-aaa", "evt-bbb"} {
		exists, err := store.Contains(context.Background(), id)
		if err != nil {
			t.Fatalf("store.Contains(%q) error: %v", id, err)
		}
		if !exists {
			t.Errorf("store.Contains(%q) = false, want true", id)
		}
	}
}

// ---------------------------------------------------------------------------
// failingIdempotencyStore: a store that always returns errors (for fail-open test).
// ---------------------------------------------------------------------------

type failingIdempotencyStore struct{}

func (f *failingIdempotencyStore) Contains(_ context.Context, _ string) (bool, error) {
	return false, errors.New("store unavailable")
}

func (f *failingIdempotencyStore) Add(_ context.Context, _ string) error {
	return errors.New("store unavailable")
}
