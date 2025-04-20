package ratelimiter

import (
	"fmt"
	"math"
	"sync"
	"time"
)

type Clock interface {
	Now() time.Time
	Sleep(d time.Duration)
}

type realClock struct{}

func (realClock) Now() time.Time        { return time.Now() }
func (realClock) Sleep(d time.Duration) { time.Sleep(d) }

// Rate defines events per second.
type Rate float64

const InfiniteRate = Rate(math.MaxFloat64)

func Every(interval time.Duration) Rate {
	if interval <= 0 {
		return InfiniteRate
	}
	return 1 / Rate(interval.Seconds())
}

// RateLimiter enforces a maximum rate and burst for events.
type RateLimiter struct {
	mu          sync.Mutex
	rate        Rate
	maxTokens   int
	tokens      float64
	updatedAt   time.Time
	eventAt     time.Time
	clock       Clock
}

func New(rate Rate, burst int, clk Clock) *RateLimiter {
	if clk == nil {
		clk = realClock{}
	}
	now := clk.Now()
	return &RateLimiter{
		rate:      rate,
		maxTokens: burst,
		tokens:    float64(burst),
		updatedAt: now,
		eventAt:   now,
		clock:     clk,
	}
}

func (rl *RateLimiter) Rate() Rate {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	return rl.rate
}

func (rl *RateLimiter) Burst() int {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	return rl.maxTokens
}

func (rl *RateLimiter) AvailableTokens() float64 {
	return rl.tokensAt(rl.clock.Now())
}

func (rl *RateLimiter) tokensAt(t time.Time) float64 {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	return rl.updateTokens(t)
}

func (rl *RateLimiter) updateTokens(t time.Time) float64 {
	if t.Before(rl.updatedAt) {
		t = rl.updatedAt
	}
	delta := rl.rate.tokensFromDuration(t.Sub(rl.updatedAt))
	tokens := rl.tokens + delta
	if max := float64(rl.maxTokens); tokens > max {
		tokens = max
	}
	return tokens
}

func (rl *RateLimiter) Allow() bool {
	return rl.AllowN(1)
}

func (rl *RateLimiter) AllowN(n int) bool {
	return rl.reserve(rl.clock.Now(), n, 0).ok
}

func (rl *RateLimiter) Wait(n int) error {
	t := rl.clock.Now()

	rl.mu.Lock()
	burst := rl.maxTokens
	rate := rl.rate
	rl.mu.Unlock()

	if n > burst && rate != InfiniteRate {
		return fmt.Errorf("rate: Wait(n=%d) exceeds limiter's burst %d", n, burst)
	}

	r := rl.reserve(t, n, InfiniteDuration)
	if !r.ok {
		return fmt.Errorf("rate: Wait(n=%d) cannot reserve tokens", n)
	}
	delay := r.DelayFrom(t)
	if delay > 0 {
		rl.clock.Sleep(delay)
	}
	return nil
}

func (rl *RateLimiter) SetRate(newRate Rate) {
	rl.SetRateAt(rl.clock.Now(), newRate)
}

func (rl *RateLimiter) SetRateAt(t time.Time, newRate Rate) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.tokens = rl.updateTokens(t)
	rl.rate = newRate
	rl.updatedAt = t
}

func (rl *RateLimiter) SetBurst(newBurst int) {
	rl.SetBurstAt(rl.clock.Now(), newBurst)
}

func (rl *RateLimiter) SetBurstAt(t time.Time, newBurst int) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.maxTokens = newBurst
	rl.tokens = rl.updateTokens(t)
	rl.updatedAt = t
	rl.eventAt = t
}

// internal reservation struct

type reservation struct {
	ok        bool
	r         *RateLimiter
	tokens    int
	timeToAct time.Time
	rate      Rate
}

const InfiniteDuration = time.Duration(math.MaxInt64)

func (r *reservation) DelayFrom(t time.Time) time.Duration {
	if !r.ok {
		return InfiniteDuration
	}
	delay := r.timeToAct.Sub(t)
	if delay < 0 {
		return 0
	}
	return delay
}

func (r *reservation) Cancel() {
	r.CancelAt(r.r.clock.Now())
}

func (r *reservation) CancelAt(t time.Time) {
	if !r.ok {
		return
	}
	r.r.mu.Lock()
	defer r.r.mu.Unlock()

	if r.r.rate == InfiniteRate || r.tokens == 0 || r.timeToAct.Before(t) {
		return
	}

	restore := float64(r.tokens) - r.rate.tokensFromDuration(r.r.eventAt.Sub(r.timeToAct))
	if restore <= 0 {
		return
	}
	tokens := r.r.updateTokens(t) + restore
	if max := float64(r.r.maxTokens); tokens > max {
		tokens = max
	}
	r.r.updatedAt = t
	r.r.tokens = tokens

	if r.timeToAct == r.r.eventAt {
		prev := r.timeToAct.Add(r.rate.durationFromTokens(float64(-r.tokens)))
		if !prev.Before(t) {
			r.r.eventAt = prev
		}
	}
}

func (rl *RateLimiter) reserve(t time.Time, n int, maxWait time.Duration) reservation {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if rl.rate == InfiniteRate {
		return reservation{ok: true, r: rl, tokens: n, timeToAct: t}
	}

	tokens := rl.updateTokens(t) - float64(n)
	var wait time.Duration
	if tokens < 0 {
		wait = rl.rate.durationFromTokens(-tokens)
	}

	ok := n <= rl.maxTokens && wait <= maxWait
	res := reservation{
		ok:    ok,
		r:     rl,
		rate:  rl.rate,
		tokens: n,
	}
	if ok {
		res.timeToAct = t.Add(wait)
		rl.updatedAt = t
		rl.tokens = tokens
		rl.eventAt = res.timeToAct
	}
	return res
}

func (r Rate) durationFromTokens(tokens float64) time.Duration {
	if r <= 0 {
		return InfiniteDuration
	}
	d := (tokens / float64(r)) * float64(time.Second)
	if d > float64(math.MaxInt64) {
		return InfiniteDuration
	}
	return time.Duration(d)
}

func (r Rate) tokensFromDuration(d time.Duration) float64 {
	if r <= 0 {
		return 0
	}
	return d.Seconds() * float64(r)
}
