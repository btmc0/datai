// Package sessionfiles provides periodic maintenance for the session
// store: purging ephemeral dead sessions that were never attributed
// to a conversation file and pruning expired dead-session history.
//
// File-backed conversation discovery is handled by the conversations
// package. This package only handles cleanup.
package sessionfiles

import (
	"log"
	"time"

	"github.com/sting8k/jump/services/jumpd/internal/store"
)

const (
	staleEphemeralMaxAge = 10 * time.Minute
	deadSessionTTL       = 7 * 24 * time.Hour
)

// Scanner provides periodic store maintenance.
type Scanner struct {
	store *store.Store

	// OnFirstScan is called once after the initial purge completes.
	// At that point the store has the full set of known sessions,
	// making it safe to clean up stale references elsewhere (e.g.
	// project session arrays).
	OnFirstScan func()

	// OnRemove is called before the scanner removes a session from the
	// store, allowing callers to clean up secondary indexes such as
	// project session arrays while the full session record is still known.
	OnRemove func(store.Session)

	now func() time.Time
}

func New(s *store.Store) *Scanner {
	return &Scanner{store: s}
}

// Run performs an initial purge, fires OnFirstScan, then purges
// periodically until stop is closed.
func (sc *Scanner) Run(interval time.Duration, stop <-chan struct{}) {
	sc.purgeOnce()
	if sc.OnFirstScan != nil {
		sc.OnFirstScan()
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			sc.purgeOnce()
		}
	}
}

// PurgeStaleSessions removes dead sessions that have no slug and
// are older than maxAge. These are short-lived sessions that exited
// without ever being attributed to a conversation file.
func (sc *Scanner) PurgeStaleSessions(maxAge time.Duration) {
	now := sc.currentTime()
	for _, s := range sc.store.List() {
		if s.Alive || s.Resumable || s.Slug != "" {
			continue
		}
		exited, err := time.Parse(time.RFC3339, s.ExitedAt)
		if err != nil {
			continue
		}
		if now.Sub(exited) > maxAge {
			log.Printf("sessionfiles: purging stale session %s (exited %s ago)", s.ID, now.Sub(exited).Round(time.Second))
			sc.removeSession(s)
		}
	}
}

// PurgeExpiredDeadSessions removes local dead sessions older than maxAge.
// Peer-owned sessions are skipped because their owning jumpd is the state owner.
func (sc *Scanner) PurgeExpiredDeadSessions(maxAge time.Duration) {
	now := sc.currentTime()
	for _, s := range sc.store.List() {
		if s.Alive || s.Peer != "" {
			continue
		}
		exited, err := time.Parse(time.RFC3339, s.ExitedAt)
		if err != nil {
			continue
		}
		age := now.Sub(exited)
		if age > maxAge {
			log.Printf("sessionfiles: pruning dead session %s (exited %s ago; ttl %s)", s.ID, age.Round(time.Second), maxAge)
			sc.removeSession(s)
		}
	}
}

func (sc *Scanner) purgeOnce() {
	sc.PurgeStaleSessions(staleEphemeralMaxAge)
	sc.PurgeExpiredDeadSessions(deadSessionTTL)
}

func (sc *Scanner) removeSession(sess store.Session) {
	if sc.OnRemove != nil {
		sc.OnRemove(sess)
	}
	sc.store.Remove(sess.ID)
}

func (sc *Scanner) currentTime() time.Time {
	if sc.now != nil {
		return sc.now().UTC()
	}
	return time.Now().UTC()
}
