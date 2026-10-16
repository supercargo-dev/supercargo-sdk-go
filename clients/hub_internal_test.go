package clients

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComputeBackoff_Bounds(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		baseDelay time.Duration
		attempt   int
	}{
		{name: "attempt 1 base 100ms", baseDelay: 100 * time.Millisecond, attempt: 1},
		{name: "attempt 2 base 100ms", baseDelay: 100 * time.Millisecond, attempt: 2},
		{name: "attempt 3 base 100ms", baseDelay: 100 * time.Millisecond, attempt: 3},
		{name: "attempt 4 base 100ms", baseDelay: 100 * time.Millisecond, attempt: 4},
		{name: "attempt 1 base 500ms", baseDelay: 500 * time.Millisecond, attempt: 1},
		{name: "attempt 2 base 500ms", baseDelay: 500 * time.Millisecond, attempt: 2},
		{name: "attempt 3 base 500ms", baseDelay: 500 * time.Millisecond, attempt: 3},
		{name: "attempt 4 base 500ms", baseDelay: 500 * time.Millisecond, attempt: 4},
		{name: "attempt 5 base 500ms (capped)", baseDelay: 500 * time.Millisecond, attempt: 5},
		{name: "attempt 6 base 500ms (capped)", baseDelay: 500 * time.Millisecond, attempt: 6},
		{name: "attempt 1 base 1s", baseDelay: 1 * time.Second, attempt: 1},
		{name: "attempt 2 base 1s", baseDelay: 1 * time.Second, attempt: 2},
		{name: "attempt 3 base 1s (capped)", baseDelay: 1 * time.Second, attempt: 3},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			shift := tt.attempt - 1
			if shift > 30 {
				shift = 30
			}
			capped := tt.baseDelay * (1 << shift)
			if capped > 5*time.Second || capped <= 0 || tt.baseDelay >= 5*time.Second {
				capped = 5 * time.Second
			}

			minDelay := time.Duration(float64(capped) * 0.5)
			maxDelay := capped

			for i := 0; i < 50; i++ {
				delay := computeBackoff(tt.baseDelay, tt.attempt)
				assert.GreaterOrEqual(t, delay, minDelay, "delay %v should be >= 0.5 * capped (%v)", delay, minDelay)
				assert.LessOrEqual(t, delay, maxDelay, "delay %v should be <= capped (%v)", delay, maxDelay)
			}
		})
	}
}

func TestComputeBackoff_CeilingVariance(t *testing.T) {
	t.Parallel()

	ceilingAttempts := []int{5, 6, 10}
	baseDelay := 500 * time.Millisecond

	for _, attempt := range ceilingAttempts {
		attempt := attempt
		t.Run(fmt.Sprintf("attempt_%d", attempt), func(t *testing.T) {
			t.Parallel()

			delays := make([]time.Duration, 100)
			minVal := time.Duration(1<<63 - 1)
			maxVal := time.Duration(0)

			for i := 0; i < 100; i++ {
				d := computeBackoff(baseDelay, attempt)
				delays[i] = d
				assert.GreaterOrEqual(t, d, 2500*time.Millisecond, "delay must be >= 2.5s")
				assert.LessOrEqual(t, d, 5000*time.Millisecond, "delay must be <= 5.0s")
				if d < minVal {
					minVal = d
				}
				if d > maxVal {
					maxVal = d
				}
			}

			assert.NotEqual(t, minVal, maxVal, "delays at ceiling must be non-constant across 100 iterations (min=%v, max=%v)", minVal, maxVal)
		})
	}
}

func TestComputeBackoff_NonPositiveAndOverflow(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		baseDelay time.Duration
		attempt   int
		wantZero  bool
		minDelay  time.Duration
		maxDelay  time.Duration
	}{
		{name: "zero base delay", baseDelay: 0, attempt: 1, wantZero: true},
		{name: "negative base delay", baseDelay: -100 * time.Millisecond, attempt: 1, wantZero: true},
		{name: "negative base delay large", baseDelay: -5 * time.Second, attempt: 3, wantZero: true},
		{name: "zero attempt treated as attempt 1", baseDelay: 100 * time.Millisecond, attempt: 0, wantZero: false, minDelay: 50 * time.Millisecond, maxDelay: 100 * time.Millisecond},
		{name: "negative attempt treated as attempt 1", baseDelay: 100 * time.Millisecond, attempt: -5, wantZero: false, minDelay: 50 * time.Millisecond, maxDelay: 100 * time.Millisecond},
		{name: "base delay equal to maxBackoffDelay", baseDelay: 5 * time.Second, attempt: 1, wantZero: false, minDelay: 2500 * time.Millisecond, maxDelay: 5000 * time.Millisecond},
		{name: "base delay greater than maxBackoffDelay", baseDelay: 10 * time.Second, attempt: 1, wantZero: false, minDelay: 2500 * time.Millisecond, maxDelay: 5000 * time.Millisecond},
		{name: "extreme attempt overflow check 62", baseDelay: 100 * time.Millisecond, attempt: 62, wantZero: false, minDelay: 2500 * time.Millisecond, maxDelay: 5000 * time.Millisecond},
		{name: "extreme attempt overflow check 63", baseDelay: 100 * time.Millisecond, attempt: 63, wantZero: false, minDelay: 2500 * time.Millisecond, maxDelay: 5000 * time.Millisecond},
		{name: "extreme attempt overflow check 100", baseDelay: 100 * time.Millisecond, attempt: 100, wantZero: false, minDelay: 2500 * time.Millisecond, maxDelay: 5000 * time.Millisecond},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			for i := 0; i < 20; i++ {
				delay := computeBackoff(tt.baseDelay, tt.attempt)
				if tt.wantZero {
					require.Equal(t, time.Duration(0), delay, "expected 0s for non-positive delay")
				} else {
					assert.GreaterOrEqual(t, delay, tt.minDelay, "delay %v should be >= %v", delay, tt.minDelay)
					assert.LessOrEqual(t, delay, tt.maxDelay, "delay %v should be <= %v", delay, tt.maxDelay)
				}
			}
		})
	}
}

func TestComputeBackoff_ConcurrentRace(t *testing.T) {
	t.Parallel()

	const goroutines = 50
	const iterations = 100
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for g := 0; g < goroutines; g++ {
		go func(id int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				attempt := (id+i)%10 + 1
				d := computeBackoff(500*time.Millisecond, attempt)
				if d < 0 || d > 5*time.Second {
					t.Errorf("unexpected delay out of range: %v", d)
				}
			}
		}(g)
	}

	wg.Wait()
}
