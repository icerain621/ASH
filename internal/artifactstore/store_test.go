package artifactstore

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestFSStorePutAndSignedURL(t *testing.T) {
	store := NewFSStore(t.TempDir())
	ref, err := store.Put(context.Background(), "runs/run_1/diff.patch", strings.NewReader("diff"), "text/plain")
	if err != nil {
		t.Fatal(err)
	}
	if ref.URI == "" || ref.SizeBytes != 4 {
		t.Fatalf("bad ref: %+v", ref)
	}
	url, err := store.SignedURL(context.Background(), ref.Key, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if url != ref.URI {
		t.Fatalf("url=%q want %q", url, ref.URI)
	}
	path := strings.TrimPrefix(ref.URI, "fs://")
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}

func TestFactoryProfiles(t *testing.T) {
	dir := t.TempDir()
	if _, ok := New("local", dir).(*FSStore); !ok {
		t.Fatal("local artifact store should resolve to FSStore")
	}
	fs := Describe("filesystem", dir)
	if fs.Kind != "fs" || !fs.Ready || fs.URI == "" {
		t.Fatalf("fs profile=%+v want ready fs", fs)
	}
	s3 := Describe("s3-compatible", dir)
	if s3.Kind != "s3-compatible" || s3.Ready || !s3.ObjectStore || s3.Error == "" {
		t.Fatalf("s3 profile=%+v want unsupported object store profile", s3)
	}
	if _, ok := New("s3-compatible", dir).(UnsupportedStore); !ok {
		t.Fatal("incomplete s3-compatible config should resolve to UnsupportedStore")
	}
}

func TestS3CompatibleStorePutAndSignedURL(t *testing.T) {
	var gotMethod, gotPath, gotContentType, gotBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotContentType = r.Header.Get("content-type")
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		gotBody = string(body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	t.Setenv("ASH_S3_ENDPOINT", server.URL)
	t.Setenv("ASH_S3_BUCKET", "ash-artifacts")
	t.Setenv("ASH_S3_REGION", "us-test-1")
	t.Setenv("ASH_S3_ACCESS_KEY_ID", "test-key")
	t.Setenv("ASH_S3_SECRET_ACCESS_KEY", "test-secret")
	t.Setenv("ASH_S3_FORCE_PATH_STYLE", "true")

	profile := Describe("s3-compatible", t.TempDir())
	if !profile.Ready || profile.URI != "s3://ash-artifacts" || !profile.ObjectStore {
		t.Fatalf("profile=%+v want ready s3-compatible", profile)
	}
	store, ok := New("s3-compatible", t.TempDir()).(*S3Store)
	if !ok {
		t.Fatalf("store type=%T want *S3Store", New("s3-compatible", t.TempDir()))
	}
	ref, err := store.Put(context.Background(), "runs/run_1/release_notes.md", strings.NewReader("notes"), "text/markdown")
	if err != nil {
		t.Fatal(err)
	}
	if ref.URI != "s3://ash-artifacts/runs/run_1/release_notes.md" || ref.SizeBytes != 5 {
		t.Fatalf("ref=%+v", ref)
	}
	if gotMethod != http.MethodPut || gotPath != "/ash-artifacts/runs/run_1/release_notes.md" {
		t.Fatalf("method/path=%s %s want PUT /ash-artifacts/runs/run_1/release_notes.md", gotMethod, gotPath)
	}
	if gotContentType != "text/markdown" || gotBody != "notes" {
		t.Fatalf("content-type/body=%q %q", gotContentType, gotBody)
	}
	url, err := store.SignedURL(context.Background(), ref.Key, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(url, server.URL+"/ash-artifacts/runs/run_1/release_notes.md?") {
		t.Fatalf("signed url=%q", url)
	}
}
