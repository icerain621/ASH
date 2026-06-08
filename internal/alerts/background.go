package alerts

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/ash-repwiki/ash/internal/store"
)

// ParseEvalInterval reads ASH_ALERTS_EVAL_INTERVAL (e.g. 5m, 1h). Empty/0/off disables.
func ParseEvalInterval(raw string) (time.Duration, bool) {
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

// StartBackgroundEvaluator periodically evaluates alert rules for known spaces.
// Returns a cancel func; no-op when interval <= 0.
func StartBackgroundEvaluator(db *store.DB, interval time.Duration) context.CancelFunc {
	ctx, cancel := context.WithCancel(context.Background())
	if db == nil || interval <= 0 {
		return cancel
	}
	svc := NewService(db)
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		run := func() {
			for _, spaceID := range listAlertSpaces(db) {
				if _, err := svc.Evaluate(spaceID); err != nil {
					log.Printf("alerts: evaluate space=%s: %v", spaceID, err)
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

func listAlertSpaces(db *store.DB) []string {
	if db == nil {
		return []string{"local"}
	}
	var ids []string
	if err := db.Model(&store.Space{}).Pluck("id", &ids).Error; err != nil || len(ids) == 0 {
		return []string{"local"}
	}
	return ids
}
