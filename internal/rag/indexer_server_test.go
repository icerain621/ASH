package rag

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ash-repwiki/ash/internal/store"
	ashv1 "github.com/ash-repwiki/ash/proto/ash/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestStartIndexerServerEmptyAddrNoop(t *testing.T) {
	rt, err := StartIndexerServer("", NewService(store.OpenTest(t, t.TempDir())))
	if err != nil {
		t.Fatal(err)
	}
	if rt != nil {
		t.Fatalf("want nil runtime for empty addr, got %+v", rt)
	}
}

func TestIndexerServerIndexRepoAndQuery(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	svc := NewService(db)
	if err := svc.ensureSQLiteFTS(); err != nil {
		t.Skipf("sqlite fts5 unavailable: %v", err)
	}

	repo := t.TempDir()
	if err := os.WriteFile(filepath.Join(repo, "note.md"), []byte("dx76 indexer loopback body\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	rt, err := StartIndexerServer("127.0.0.1:0", svc)
	if err != nil {
		t.Fatal(err)
	}
	defer rt.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	conn, err := grpc.DialContext(ctx, rt.Addr, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	client := ashv1.NewRagIndexerServiceClient(conn)

	idx, err := client.IndexRepo(ctx, &ashv1.IndexRepoRequest{
		RepoRoot: repo,
		Context:  &ashv1.TraceContext{SpaceId: "space_dx76"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if idx.GetStatus().GetCode() != "OK" || idx.GetDocuments() < 1 {
		t.Fatalf("index=%+v want OK with documents", idx)
	}

	q, err := client.Query(ctx, &ashv1.QueryRequest{
		RepoRoot: repo,
		Text:     "dx76 indexer loopback",
		TopK:     3,
		Context:  &ashv1.TraceContext{SpaceId: "space_dx76"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if q.GetStatus().GetCode() != "OK" || len(q.GetItems()) < 1 {
		t.Fatalf("query=%+v want hits", q)
	}
}

func TestIndexerServerRejectsEmptyQueryText(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	lis := mustListen(t)
	srv := grpc.NewServer()
	ashv1.RegisterRagIndexerServiceServer(srv, NewIndexerServer(NewService(db)))
	go func() { _ = srv.Serve(lis) }()
	defer srv.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := grpc.DialContext(ctx, lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	resp, err := ashv1.NewRagIndexerServiceClient(conn).Query(ctx, &ashv1.QueryRequest{Text: "  "})
	if err != nil {
		t.Fatal(err)
	}
	if resp.GetStatus().GetCode() != "INVALID_REQUEST" {
		t.Fatalf("status=%+v want INVALID_REQUEST", resp.GetStatus())
	}
}

func mustListen(t *testing.T) net.Listener {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	return lis
}
