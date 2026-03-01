package database

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRetryBackoff_ExponentialWithJitter(t *testing.T) {
	// Verify the base durations are approximately 1s, 2s, 4s with ±25% jitter.
	for attempt := 0; attempt < 3; attempt++ {
		base := defaultRetryBaseWait << attempt // 1s, 2s, 4s
		minExpected := time.Duration(float64(base) * (1 - retryJitterFraction))
		maxExpected := time.Duration(float64(base) * (1 + retryJitterFraction))

		for i := 0; i < 20; i++ {
			d := retryBackoff(attempt)
			assert.GreaterOrEqual(t, d, minExpected, "attempt %d iteration %d: %v < %v", attempt, i, d, minExpected)
			assert.LessOrEqual(t, d, maxExpected, "attempt %d iteration %d: %v > %v", attempt, i, d, maxExpected)
		}
	}
}

func TestRetryBackoff_IncreasingDurations(t *testing.T) {
	// Average of many samples should show increasing trend.
	var sums [3]time.Duration
	const n = 100
	for attempt := 0; attempt < 3; attempt++ {
		for i := 0; i < n; i++ {
			sums[attempt] += retryBackoff(attempt)
		}
	}
	assert.Less(t, sums[0], sums[1], "attempt 0 avg should be less than attempt 1 avg")
	assert.Less(t, sums[1], sums[2], "attempt 1 avg should be less than attempt 2 avg")
}

func TestIsConnectionError(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		expect bool
	}{
		{"nil error", nil, false},
		{"connection refused", assert.AnError, false},
	}
	// The function checks string patterns, so we test with error strings.
	_ = tests

	assert.False(t, isConnectionError(nil))
	assert.True(t, isConnectionError(errStr("dial tcp 127.0.0.1:5432: connection refused")))
	assert.True(t, isConnectionError(errStr("connection reset by peer")))
	assert.True(t, isConnectionError(errStr("broken pipe")))
	assert.True(t, isConnectionError(errStr("i/o timeout")))
	assert.True(t, isConnectionError(errStr("EOF")))
	assert.True(t, isConnectionError(errStr("could not connect to server")))
	assert.False(t, isConnectionError(errStr("syntax error at or near")))
	assert.False(t, isConnectionError(errStr("duplicate key value violates unique constraint")))
	assert.False(t, isConnectionError(errStr("relation does not exist")))
}

type errStr string

func (e errStr) Error() string { return string(e) }
