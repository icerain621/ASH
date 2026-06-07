package runs

import (
	"context"
	"testing"

	"github.com/ash-repwiki/ash/internal/events"
	"github.com/ash-repwiki/ash/internal/store"
)

func TestEventsForWithContextDoesNotRecurse(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	ev := events.NewService(db)
	svc := NewService(db, ev, nil, nil)
	ctx := context.Background()
	bound := svc.WithContext(ctx)

	got := bound.eventsFor()
	if got == nil {
		t.Fatal("eventsFor returned nil")
	}
	if got == ev {
		t.Fatal("expected context-bound events service, got base pointer")
	}
	if got.WithContext(ctx) == got {
		// sanity: WithContext on bound service is stable
	}
}
