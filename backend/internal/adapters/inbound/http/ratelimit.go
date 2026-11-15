package httpapi

import (
	"sync"
	"time"
)

// Default login rate-limit policy, applied separately to the client IP and to
// the username.
const (
	DefaultLoginMaxFailures = 10
	DefaultLoginWindow      = 15 * time.Minute
	DefaultLoginMaxEntries  = 10_000
)

// inFlightRetry is the Retry-After suggested when a key is refused only
// because attempts are still in flight (they may yet succeed).
const inFlightRetry = time.Second

// LoginLimiter counts failed logins per key in a fixed window and blocks the
// key once maxFailures is reached until the window ends.
//
// Checks are atomic: Acquire reserves an attempt before the password is
// verified and counts it against the limit (failures + in-flight attempts),
// so a concurrent burst cannot exceed maxFailures. Each Attempt must be
// finished with exactly one of Fail, Succeed or Release.
//
// Memory is bounded by maxEntries: when full, expired entries are pruned and,
// if still full, the oldest idle entry (or, failing that, the oldest entry)
// is evicted.
type LoginLimiter struct {
	mu          sync.Mutex
	maxFailures int
	window      time.Duration
	maxEntries  int
	now         func() time.Time
	entries     map[string]*limitEntry
}

type limitEntry struct {
	failures int
	inFlight int
	start    time.Time
}

// NewLoginLimiter returns a limiter. now may be nil (time.Now).
func NewLoginLimiter(maxFailures int, window time.Duration, maxEntries int, now func() time.Time) *LoginLimiter {
	if now == nil {
		now = time.Now
	}
	return &LoginLimiter{
		maxFailures: max(maxFailures, 1),
		window:      window,
		maxEntries:  max(maxEntries, 1),
		now:         now,
		entries:     make(map[string]*limitEntry),
	}
}

// DefaultLoginLimiter returns a limiter with the default policy
// (10 failures per 15 minutes per key, at most 10k tracked keys).
func DefaultLoginLimiter() *LoginLimiter {
	return NewLoginLimiter(DefaultLoginMaxFailures, DefaultLoginWindow, DefaultLoginMaxEntries, nil)
}

// Attempt is a reserved login attempt for a set of keys.
type Attempt struct {
	l    *LoginLimiter
	keys []string
	once sync.Once
}

// Acquire reserves one attempt for every key. If any key is at its limit
// (failures plus in-flight attempts), nothing is reserved and Acquire returns
// false with how long to wait before retrying.
func (l *LoginLimiter) Acquire(keys ...string) (*Attempt, time.Duration, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	var retry time.Duration
	for _, k := range keys {
		e := l.live(k, now)
		if e == nil || e.failures+e.inFlight < l.maxFailures {
			continue
		}
		wait := inFlightRetry
		if e.failures >= l.maxFailures {
			wait = e.start.Add(l.window).Sub(now)
		}
		retry = max(retry, wait)
	}
	if retry > 0 {
		return nil, retry, false
	}
	for _, k := range keys {
		e := l.live(k, now)
		if e == nil {
			l.makeRoom(now)
			e = &limitEntry{start: now}
			l.entries[k] = e
		}
		e.inFlight++
	}
	return &Attempt{l: l, keys: keys}, 0, true
}

// Fail ends the attempt as a failed login: every key records a failure.
func (a *Attempt) Fail() {
	a.finish(func(e *limitEntry, _ string) { e.failures++ })
}

// Succeed ends the attempt as a successful login and clears the failure
// count of the given keys (which should be a subset of the attempt's keys).
// Other keys keep their counts.
func (a *Attempt) Succeed(reset ...string) {
	a.finish(func(e *limitEntry, k string) {
		for _, r := range reset {
			if r == k {
				e.failures = 0
			}
		}
	})
}

// Release ends the attempt without counting it (e.g. a server error).
func (a *Attempt) Release() {
	a.finish(func(*limitEntry, string) {})
}

func (a *Attempt) finish(update func(*limitEntry, string)) {
	a.once.Do(func() {
		l := a.l
		l.mu.Lock()
		defer l.mu.Unlock()
		for _, k := range a.keys {
			e, ok := l.entries[k]
			if !ok {
				continue // evicted meanwhile
			}
			e.inFlight = max(e.inFlight-1, 0)
			update(e, k)
			if e.failures == 0 && e.inFlight == 0 {
				delete(l.entries, k)
			}
		}
	})
}

// Len returns the number of tracked keys.
func (l *LoginLimiter) Len() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.entries)
}

// live returns the entry for key, or nil if there is none. An expired entry
// is deleted, or restarted if attempts are still in flight. Caller holds mu.
func (l *LoginLimiter) live(key string, now time.Time) *limitEntry {
	e, ok := l.entries[key]
	if !ok {
		return nil
	}
	if l.expired(e, now) {
		if e.inFlight == 0 {
			delete(l.entries, key)
			return nil
		}
		e.failures, e.start = 0, now
	}
	return e
}

func (l *LoginLimiter) expired(e *limitEntry, now time.Time) bool {
	return now.Sub(e.start) >= l.window
}

// makeRoom ensures there is space for one more entry. Caller holds mu.
func (l *LoginLimiter) makeRoom(now time.Time) {
	if len(l.entries) < l.maxEntries {
		return
	}
	for k, e := range l.entries {
		if l.expired(e, now) && e.inFlight == 0 {
			delete(l.entries, k)
		}
	}
	for len(l.entries) >= l.maxEntries {
		// Prefer evicting idle entries so in-flight reservations stay counted.
		var oldestKey string
		var oldest *limitEntry
		for k, e := range l.entries {
			if oldest == nil || (oldest.inFlight > 0 && e.inFlight == 0) ||
				((oldest.inFlight > 0) == (e.inFlight > 0) && e.start.Before(oldest.start)) {
				oldestKey, oldest = k, e
			}
		}
		delete(l.entries, oldestKey)
	}
}
