//go:build integration

package rag

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ash-repwiki/ash/internal/store"
)

func TestPostgresRAGFTSQuery(t *testing.T) {
	pgURL := os.Getenv("ASH_DATABASE_URL")
	if pgURL == "" {
		t.Skip("ASH_DATABASE_URL unset")
	}
	if os.Getenv("ASH_MIGRATE_E2E") != "1" {
		t.Skip("set ASH_MIGRATE_E2E=1 for live postgres rag fts test")
	}
	t.Setenv("ASH_SCHEMA_MODE", "sql")

	dir := t.TempDir()
	db, err := store.OpenWithDatabaseURL(dir, pgURL)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if db.Dialect() != "postgres" {
		t.Fatalf("dialect=%q want postgres", db.Dialect())
	}

	svc := NewService(db)
	repo := t.TempDir()
	content := "postgres tsvector retrieval evidence\n"
	if err := os.WriteFile(filepath.Join(repo, "pg.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Index(IndexRequest{RepoRoot: repo, SpaceID: "local"}); err != nil {
		t.Fatal(err)
	}
	if svc.FtsEngine() != "postgres-tsvector" {
		t.Fatalf("engine=%q want postgres-tsvector", svc.FtsEngine())
	}
	resp, err := svc.Query(QueryRequest{RepoRoot: repo, Text: "tsvector evidence", TopK: 3, SpaceID: "local"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.RetrievalMode != RetrievalModeFTS || len(resp.Items) == 0 || resp.Items[0].Path != "pg.md" {
		t.Fatalf("resp=%+v want fts hit on pg.md", resp)
	}
}
