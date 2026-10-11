package agenttpl

import (
	"encoding/json"
	"sync"
)

// ApprovedLoader resolves an approved custom template from durable storage (G01).
type ApprovedLoader func(id string) (Manifest, bool)

var (
	loaderMu sync.RWMutex
	loader   ApprovedLoader
)

// SetApprovedLoader installs a durable lookup used by GetProduction after the in-process catalog.
func SetApprovedLoader(fn ApprovedLoader) {
	loaderMu.Lock()
	loader = fn
	loaderMu.Unlock()
}

// ClearApprovedLoaderForTest clears the durable loader (tests only).
func ClearApprovedLoaderForTest() {
	SetApprovedLoader(nil)
}

func loadApprovedPersistent(id string) (Manifest, bool) {
	loaderMu.RLock()
	fn := loader
	loaderMu.RUnlock()
	if fn == nil {
		return Manifest{}, false
	}
	return fn(id)
}

// ManifestFromJSON decodes a stored manifest payload.
func ManifestFromJSON(raw string) (Manifest, error) {
	var m Manifest
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return Manifest{}, err
	}
	return Load(m)
}
