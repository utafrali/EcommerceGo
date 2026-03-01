package kafka

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// IdempotencyStore is the interface for checking and storing processed event IDs.
// Implementations must be safe for concurrent use.
type IdempotencyStore interface {
	// Contains returns true if the event ID has already been processed.
	Contains(ctx context.Context, eventID string) (bool, error)
	// Add marks an event ID as processed. It should be called after successful processing.
	Add(ctx context.Context, eventID string) error
}

// MemoryIdempotencyStore is an in-memory implementation of IdempotencyStore.
// Suitable for development and single-instance deployments. Entries expire
// after the configured TTL to bound memory usage.
type MemoryIdempotencyStore struct {
	mu      sync.RWMutex
	entries map[string]time.Time
	ttl     time.Duration
}

// NewMemoryIdempotencyStore creates a new in-memory idempotency store with the
// given TTL. Expired entries are lazily cleaned up on access.
func NewMemoryIdempotencyStore(ttl time.Duration) *MemoryIdempotencyStore {
	return &MemoryIdempotencyStore{
		entries: make(map[string]time.Time),
		ttl:     ttl,
	}
}

// Contains checks if the event ID exists and is not expired.
func (s *MemoryIdempotencyStore) Contains(_ context.Context, eventID string) (bool, error) {
	s.mu.RLock()
	ts, exists := s.entries[eventID]
	s.mu.RUnlock()

	if !exists {
		return false, nil
	}

	// Lazily expire old entries.
	if time.Since(ts) > s.ttl {
		s.mu.Lock()
		delete(s.entries, eventID)
		s.mu.Unlock()
		return false, nil
	}

	return true, nil
}

// Add marks the event ID as processed with the current timestamp.
func (s *MemoryIdempotencyStore) Add(_ context.Context, eventID string) error {
	s.mu.Lock()
	s.entries[eventID] = time.Now()
	s.mu.Unlock()
	return nil
}

// Len returns the number of entries in the store (including potentially expired ones).
func (s *MemoryIdempotencyStore) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.entries)
}

// IdempotentHandler wraps a Handler with deduplication logic. If the event's
// EventID has already been processed (according to the store), the message is
// skipped and nil is returned.
func IdempotentHandler(store IdempotencyStore, inner Handler, logger *slog.Logger) Handler {
	return func(ctx context.Context, event *Event) error {
		if event.EventID == "" {
			// No event ID — cannot deduplicate, pass through.
			return inner(ctx, event)
		}

		exists, err := store.Contains(ctx, event.EventID)
		if err != nil {
			logger.Warn("idempotency store lookup failed, processing anyway",
				slog.String("event_id", event.EventID),
				slog.String("error", err.Error()),
			)
			// On store failure, process the message rather than risk data loss.
			return inner(ctx, event)
		}

		if exists {
			logger.Debug("skipping duplicate event",
				slog.String("event_id", event.EventID),
				slog.String("event_type", event.EventType),
				slog.String("aggregate_id", event.AggregateID),
			)
			return nil
		}

		// Process the message.
		if err := inner(ctx, event); err != nil {
			return err
		}

		// Mark as processed only after successful handling.
		if addErr := store.Add(ctx, event.EventID); addErr != nil {
			logger.Warn("failed to record event ID in idempotency store",
				slog.String("event_id", event.EventID),
				slog.String("error", addErr.Error()),
			)
		}

		return nil
	}
}
