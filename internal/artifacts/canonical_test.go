package artifacts

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestMarshalCanonicalJSONStableKeyOrder(t *testing.T) {
	a := map[string]any{"z": 1, "a": map[string]any{"y": true, "b": "x"}, "m": []any{2, 1}}
	b := map[string]any{"m": []any{2, 1}, "a": map[string]any{"b": "x", "y": true}, "z": 1}
	ca, err := MarshalCanonicalJSON(a)
	if err != nil {
		t.Fatal(err)
	}
	cb, err := MarshalCanonicalJSON(b)
	if err != nil {
		t.Fatal(err)
	}
	if string(ca) != string(cb) {
		t.Fatalf("canonical mismatch:\n%s\n%s", ca, cb)
	}
	want := `{"a":{"b":"x","y":true},"m":[2,1],"z":1}`
	if string(ca) != want {
		t.Fatalf("got %s want %s", ca, want)
	}
	da, err := DigestCanonicalJSON(a)
	if err != nil {
		t.Fatal(err)
	}
	db, err := DigestCanonicalJSON(b)
	if err != nil {
		t.Fatal(err)
	}
	if da != db || da == "" {
		t.Fatalf("digest a=%q b=%q", da, db)
	}
	for i := 0; i < 50; i++ {
		again, err := DigestCanonicalJSON(a)
		if err != nil || again != da {
			t.Fatalf("iter %d digest=%q want %q err=%v", i, again, da, err)
		}
	}
}

func TestEnsureTestReportIsCanonical(t *testing.T) {
	dir := t.TempDir()
	if err := ensureTestReport(dir); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "test_report.json"))
	if err != nil {
		t.Fatal(err)
	}
	var node any
	if err := json.Unmarshal(raw, &node); err != nil {
		t.Fatal(err)
	}
	canon, err := MarshalCanonicalJSON(node)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != string(canon) {
		t.Fatalf("test_report not canonical:\n%s\n%s", raw, canon)
	}
}
