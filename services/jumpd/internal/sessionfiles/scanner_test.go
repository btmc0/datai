package sessionfiles

import (
	"testing"
	"time"

	"github.com/sting8k/jump/services/jumpd/internal/store"
)

func TestPurgeStaleSessions(t *testing.T) {
	now := time.Date(2026, 5, 17, 12, 0, 0, 0, time.UTC)
	s := store.New()

	s.Upsert(store.Session{
		ID:       "stale",
		Alive:    false,
		ExitedAt: now.Add(-2 * time.Hour).Format(time.RFC3339),
	})
	s.Upsert(store.Session{
		ID:       "fresh",
		Alive:    false,
		ExitedAt: now.Add(-10 * time.Minute).Format(time.RFC3339),
	})
	s.Upsert(store.Session{
		ID:       "resumable",
		Alive:    false,
		Slug:     "some-key",
		ExitedAt: now.Add(-2 * time.Hour).Format(time.RFC3339),
	})

	sc := New(s)
	sc.now = func() time.Time { return now }
	sc.PurgeStaleSessions(1 * time.Hour)

	ids := sessionIDs(s)
	if ids["stale"] {
		t.Error("stale session should have been purged")
	}
	if !ids["fresh"] {
		t.Error("fresh session should still be present")
	}
	if !ids["resumable"] {
		t.Error("resumable session should still be present")
	}
}

func TestPurgeExpiredDeadSessions(t *testing.T) {
	now := time.Date(2026, 5, 17, 12, 0, 0, 0, time.UTC)
	s := store.New()

	s.Upsert(store.Session{
		ID:       "old-dead-resumable",
		Alive:    false,
		Command:  []string{"bash"},
		Slug:     "old-key",
		ExitedAt: now.Add(-(deadSessionTTL + time.Minute)).Format(time.RFC3339),
	})
	s.Upsert(store.Session{
		ID:       "fresh-dead",
		Alive:    false,
		Command:  []string{"bash"},
		Slug:     "fresh-key",
		ExitedAt: now.Add(-(deadSessionTTL - time.Minute)).Format(time.RFC3339),
	})
	s.Upsert(store.Session{
		ID:       "alive-old",
		Alive:    true,
		ExitedAt: now.Add(-(deadSessionTTL + time.Hour)).Format(time.RFC3339),
	})
	s.Upsert(store.Session{
		ID:       "peer-old-dead",
		Peer:     "remote-node",
		Alive:    false,
		ExitedAt: now.Add(-(deadSessionTTL + time.Hour)).Format(time.RFC3339),
	})
	s.Upsert(store.Session{
		ID:       "unknown-exit-time",
		Alive:    false,
		ExitedAt: "",
	})

	var removed []store.Session
	sc := New(s)
	sc.now = func() time.Time { return now }
	sc.OnRemove = func(sess store.Session) {
		removed = append(removed, sess)
	}
	sc.PurgeExpiredDeadSessions(deadSessionTTL)

	ids := sessionIDs(s)
	if ids["old-dead-resumable"] {
		t.Error("old dead resumable session should have been pruned")
	}
	for _, id := range []string{"fresh-dead", "alive-old", "peer-old-dead", "unknown-exit-time"} {
		if !ids[id] {
			t.Errorf("%s should still be present", id)
		}
	}
	if len(removed) != 1 || removed[0].ID != "old-dead-resumable" || removed[0].Slug != "old-key" {
		t.Fatalf("OnRemove sessions = %#v, want old-dead-resumable with slug", removed)
	}
}

func sessionIDs(s *store.Store) map[string]bool {
	ids := map[string]bool{}
	for _, sess := range s.List() {
		ids[sess.ID] = true
	}
	return ids
}
