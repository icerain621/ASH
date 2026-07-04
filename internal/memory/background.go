package memory

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/ash-repwiki/ash/internal/events"
	"github.com/ash-repwiki/ash/internal/store"
)

// ParseSweepInterval reads ASH_MEMORY_TTL_SWEEP_INTERVAL (e.g. 1h, 24h). Empty/0/off disables.
func ParseSweepInterval(raw string) (time.Duration, bool) {
	raw = strings.TrimSpace(strings.ToLower(raw))
	switch raw {
	case "", "0", "off", "false", "disable", "disabled":
		return 0, false
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d < time.Minute {
		return 0, false
	}
	return d, true
}

// StartBackgroundTTLSweep periodically deprecates expired approved memory records per space.
// Returns a cancel func; no-op when interval <= 0.
func StartBackgroundTTLSweep(db *store.DB, interval time.Duration) context.CancelFunc {
	ctx, cancel := context.WithCancel(context.Background())
	if db == nil || interval <= 0 {
		return cancel
	}
	ev := events.NewService(db)
	svc := NewService(db, ev)
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		run := func() {
			for _, spaceID := range listMemorySpaces(db) {
				resp, err := svc.SweepTTL(SweepTTLRequest{SpaceID: spaceID, ActorID: "ash-worker-ttl"})
				if err != nil {
					log.Printf("memory ttl: sweep space=%s: %v", spaceID, err)
					continue
				}
				if resp.Deprecated > 0 {
					log.Printf("memory ttl: sweep space=%s deprecated=%d reviewDue=%d", spaceID, resp.Deprecated, resp.ReviewDue)
				}
			}
		}
		run()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				run()
			}
		}
	}()
	return cancel
}

func listMemorySpaces(db *store.DB) []string {
	if db == nil {
		return []string{"local"}
	}
	var ids []string
	if err := db.Model(&store.Space{}).Pluck("id", &ids).Error; err != nil || len(ids) == 0 {
		return []string{"local"}
	}
	return ids
}
