package ratelimiter

import (
	"testing"
	"time"
)

type fakeClock struct {
	time time.Time
}

func (fc *fakeClock) Now() time.Time {
	return fc.time
}

func (fc *fakeClock) Sleep(d time.Duration) {
	fc.time = fc.time.Add(d)
}

func newFakeClock(start time.Time) *fakeClock {
	return &fakeClock{time: start}
}

func TestAllow(t *testing.T) {
	clk := newFakeClock(time.Unix(0, 0))
	rl := New(Every(100*time.Millisecond), 2, clk)

	if !rl.Allow() {
		t.Errorf("expected first token to be allowed")
	}
	if !rl.Allow() {
		t.Errorf("expected second token to be allowed")
	}
	if rl.Allow() {
		t.Errorf("expected third token to be denied")
	}

	clk.Sleep(100 * time.Millisecond)
	if !rl.Allow() {
		t.Errorf("expected token to be allowed after 100ms")
	}
}

func TestAllowN(t *testing.T) {
	clk := newFakeClock(time.Unix(0, 0))
	rl := New(Every(200*time.Millisecond), 3, clk)
	if !rl.AllowN(3) {
		t.Errorf("expected 3 tokens to be allowed initially")
	}
	if rl.AllowN( 1) {
		t.Errorf("expected token to be denied")
	}

	clk.Sleep(600 * time.Millisecond)
	if !rl.AllowN(3) {
		t.Errorf("expected 3 tokens to be allowed after sleep")
	}
}

func TestSetRate(t *testing.T) {
	clk := newFakeClock(time.Unix(0, 0))
	rl := New(Every(200*time.Millisecond), 5, clk)
	for i := 0; i < 5; i++ {
		rl.Allow()
	}
	if rl.Allow() {
		t.Errorf("expected sixth token to be denied before rate change")
	}
	clk.Sleep(200 * time.Millisecond)
	if !rl.Allow() {
		t.Errorf("expected one token to be available after sleep")
	}

	rl.SetRate(Every(100 * time.Millisecond))
	clk.Sleep(400 * time.Millisecond)
	count := 0
	for i := 0; i < 5; i++ {
		if rl.Allow() {
			count++
		}
	}
	if count < 3 {
		t.Errorf("expected more tokens after rate increase, got %d", count)
	}
}

func TestSetBurst(t *testing.T) {
	clk := newFakeClock(time.Unix(0, 0))
	rl := New(Every(100*time.Millisecond), 1, clk)
	if !rl.Allow() {
		t.Fatalf("expected token to be allowed")
	}
	if rl.Allow() {
		t.Fatalf("expected token to be denied due to burst")
	}

	rl.SetBurst(5)
	clk.Sleep(500 * time.Millisecond)
	count := 0
	for i := 0; i < 5; i++ {
		if rl.Allow() {
			count++
		}
	}
	if count != 5 {
		t.Fatalf("expected 5 tokens after burst increase, got %d", count)
	}
}

func TestWait(t *testing.T) {
	clk := newFakeClock(time.Unix(0, 0))
	rl := New(Every(100*time.Millisecond), 2, clk)
	if err := rl.Wait(2); err != nil {
		t.Fatalf("expected wait for 2 tokens to succeed")
	}
	if err := rl.Wait(3); err == nil {
		t.Fatalf("expected wait for 3 tokens to fail (burst exceeded)")
	}
}

func TestTokensAndAdvance(t *testing.T) {
	clk := newFakeClock(time.Unix(0, 0))
	rl := New(Every(100*time.Millisecond), 3, clk)
	if tok := rl.AvailableTokens(); tok != 3 {
		t.Fatalf("expected 3 tokens initially, got %f", tok)
	}
	_ = rl.AllowN( 2)
	if tok := rl.AvailableTokens(); tok > 1.01 {
		t.Fatalf("expected about 1 token left, got %f", tok)
	}
	clk.Sleep(200 * time.Millisecond)
	if tok := rl.AvailableTokens(); tok < 2.9 {
		t.Fatalf("expected token refill after sleep, got %f", tok)
	}
}

func TestInfRate(t *testing.T) {
	clk := newFakeClock(time.Unix(0, 0))
	rl := New(InfiniteRate, 100, clk)
	if !rl.AllowN(100000) {
		t.Fatalf("expected all tokens to be allowed with Inf rate")
	}
}
