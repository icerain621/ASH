package store

import (
	"path/filepath"
	"testing"
	"time"
)

func TestMigratorCopySQLiteToSQLite(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "source.db")
	dstPath := filepath.Join(dir, "target.db")

	src, err := OpenSQLite(dir, srcPath)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if err := src.Create(&Org{ID: "org_migrate", Name: "Migrate Org", Slug: "migrate", CreatedAt: now, UpdatedAt: now}).Error; err != nil {
		t.Fatal(err)
	}
	if err := src.Create(&RunRecord{
		ID: "run_migrate", TraceID: "trc_migrate", ScenarioName: "feature_delivery",
		ScenarioVersion: "1.0.0", PolicyProfile: "default", Status: "finished",
		SpaceID: "local", StartedAt: now, CreatedAt: now, UpdatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}
	_ = src.Close()

	dst, err := OpenSQLite(dir, dstPath)
	if err != nil {
		t.Fatal(err)
	}
	_ = dst.Close()

	m := &Migrator{
		Source: mustOpenSQLite(t, dir, srcPath),
		Target: mustOpenSQLite(t, dir, dstPath),
	}
	defer m.Close()

	report, err := m.Copy(CopyOptions{BatchSize: 50})
	if err != nil {
		t.Fatal(err)
	}
	if report.TotalCopied < 2 {
		t.Fatalf("copied=%d want >=2", report.TotalCopied)
	}

	var orgCount int64
	if err := m.Target.Model(&Org{}).Count(&orgCount).Error; err != nil {
		t.Fatal(err)
	}
	if orgCount != 1 {
		t.Fatalf("orgs=%d want 1", orgCount)
	}
}

func TestMigratorPlanAndSyncState(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "a.db")
	dstPath := filepath.Join(dir, "b.db")
	src := mustOpenSQLite(t, dir, srcPath)
	now := time.Now().UTC()
	_ = src.Create(&Space{ID: "space_sync", OrgID: "org_x", Name: "S", Slug: "s", CreatedAt: now, UpdatedAt: now}).Error
	_ = src.Close()
	dst := mustOpenSQLite(t, dir, dstPath)
	_ = dst.Close()

	m := &Migrator{Source: mustOpenSQLite(t, dir, srcPath), Target: mustOpenSQLite(t, dir, dstPath)}
	defer m.Close()

	plan, err := m.Plan()
	if err != nil {
		t.Fatal(err)
	}
	if plan.SourceDialect != "sqlite" || plan.TargetDialect != "sqlite" {
		t.Fatalf("plan=%+v", plan)
	}

	if _, err := m.Copy(CopyOptions{}); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Verify(); err != nil {
		t.Fatal(err)
	}

	state, err := LoadSyncState(dir)
	if err != nil {
		t.Fatal(err)
	}
	if state.LastSyncAt != nil {
		t.Fatal("expected empty initial sync state")
	}
	if _, err := m.Sync(dir, CopyOptions{}); err != nil {
		t.Fatal(err)
	}
	state, err = LoadSyncState(dir)
	if err != nil || state.LastSyncAt == nil {
		t.Fatalf("state=%+v err=%v", state, err)
	}
	if state.LastError != "" || state.LastErrorAt != nil {
		t.Fatalf("expected cleared sync error, state=%+v", state)
	}
}

func TestMigratorSyncPersistsFailure(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "a.db")
	dstPath := filepath.Join(dir, "b.db")
	src := mustOpenSQLite(t, dir, srcPath)
	now := time.Now().UTC()
	_ = src.Create(&Space{ID: "space_fail", OrgID: "org_x", Name: "S", Slug: "s", CreatedAt: now, UpdatedAt: now}).Error
	_ = src.Close()
	dst := mustOpenSQLite(t, dir, dstPath)
	_ = dst.Close()

	m := &Migrator{Source: mustOpenSQLite(t, dir, srcPath), Target: mustOpenSQLite(t, dir, dstPath)}
	defer m.Close()

	if _, err := m.Sync(dir, CopyOptions{}); err != nil {
		t.Fatal(err)
	}
	state, err := LoadSyncState(dir)
	if err != nil || state.LastSyncAt == nil {
		t.Fatalf("state=%+v err=%v", state, err)
	}
	firstSync := *state.LastSyncAt

	if err := m.Target.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Sync(dir, CopyOptions{}); err == nil {
		t.Fatal("expected sync error with closed target")
	}
	state, err = LoadSyncState(dir)
	if err != nil {
		t.Fatal(err)
	}
	if state.LastError == "" || state.LastErrorAt == nil {
		t.Fatalf("expected persisted sync error, state=%+v", state)
	}
	if state.LastSyncAt == nil || !state.LastSyncAt.Equal(firstSync) {
		t.Fatalf("lastSyncAt=%v want %v", state.LastSyncAt, firstSync)
	}
}

func TestSyncStateRecordSuccessClearsError(t *testing.T) {
	now := time.Now().UTC()
	state := &SyncState{
		LastError:   "previous failure",
		LastErrorAt: &now,
		UpdatedAt:   now,
	}
	state.recordSuccess(now.Add(time.Minute))
	if state.LastError != "" || state.LastErrorAt != nil {
		t.Fatalf("expected cleared error, state=%+v", state)
	}
	if state.LastSyncAt == nil {
		t.Fatal("expected lastSyncAt")
	}
}

func TestDualWriteConfigRoundTrip(t *testing.T) {
	dir := t.TempDir()
	cfg := &DualWriteConfig{
		Enabled: true, PostgresURL: "postgres://u:p@127.0.0.1/ash",
		SQLitePath: filepath.Join(dir, "ash.db"), EnabledAt: time.Now().UTC(),
	}
	if err := SaveDualWriteConfig(dir, cfg); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadDualWriteConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !loaded.Enabled || loaded.PostgresURL != cfg.PostgresURL {
		t.Fatalf("loaded=%+v", loaded)
	}
}

func TestSchemaMetaSourceKeysMatchAllowsPostgresExtras(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "src.db")
	dstPath := filepath.Join(dir, "dst.db")
	src := mustOpenSQLite(t, dir, srcPath)
	dst := mustOpenSQLite(t, dir, dstPath)
	now := time.Now().UTC()
	extra := SchemaMeta{Key: "memory_schema_catalog_version", Value: "2", UpdatedAt: now}
	if err := src.Create(&extra).Error; err != nil {
		t.Fatal(err)
	}
	if err := dst.Create(&extra).Error; err != nil {
		t.Fatal(err)
	}
	if err := dst.Create(&SchemaMeta{Key: "sql_migrations", Value: "golang-migrate/v1", UpdatedAt: now}).Error; err != nil {
		t.Fatal(err)
	}

	ok, err := schemaMetaSourceKeysMatch(src.DB, dst.DB)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected source keys to match on target with extras")
	}
	if err := src.Close(); err != nil {
		t.Fatal(err)
	}
	if err := dst.Close(); err != nil {
		t.Fatal(err)
	}
}

func mustOpenSQLite(t *testing.T, dir, path string) *DB {
	t.Helper()
	db, err := OpenSQLite(dir, path)
	if err != nil {
		t.Fatal(err)
	}
	return db
}
