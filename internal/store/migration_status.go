package store

import (
	"time"
)

// MigrationSnapshot summarizes CLI migration state for ops APIs and consoles.
type MigrationSnapshot struct {
	SQLitePath          string          `json:"sqlitePath"`
	MigrationTableCount int             `json:"migrationTableCount"`
	DualWriteEnabled    bool            `json:"dualWriteEnabled"`
	DualWriteRuntime    bool            `json:"dualWriteRuntime"`
	DualWriteSource     DualWriteSource `json:"dualWriteSource,omitempty"`
	LastSyncAt          *time.Time      `json:"lastSyncAt,omitempty"`
}

// MigrationSnapshotFor builds migration/dual-write status for a data directory.
func MigrationSnapshotFor(dataDir string) (MigrationSnapshot, error) {
	shadowURL, source := ResolveDualWritePostgresURL(dataDir)
	snap := MigrationSnapshot{
		SQLitePath:          DefaultSQLitePath(dataDir),
		MigrationTableCount: MigrationCatalogSize(),
		DualWriteRuntime:    shadowURL != "",
		DualWriteSource:     source,
	}
	cfg, err := LoadDualWriteConfig(dataDir)
	if err != nil {
		return snap, err
	}
	snap.DualWriteEnabled = cfg.Enabled
	state, err := LoadSyncState(dataDir)
	if err != nil {
		return snap, err
	}
	snap.LastSyncAt = state.LastSyncAt
	return snap, nil
}
