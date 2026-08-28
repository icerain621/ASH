package diffparse

import (
	"testing"
)

func TestParseUnified(t *testing.T) {
	raw := `diff --git a/README.md b/README.md
--- a/README.md
+++ b/README.md
@@ -1,3 +1,4 @@
 line1
-line2
+line2b
+line2c
 line3
`
	files := ParseUnified(raw)
	if len(files) != 1 {
		t.Fatalf("files=%d", len(files))
	}
	if files[0].Path != "README.md" {
		t.Fatalf("path=%q", files[0].Path)
	}
	if len(files[0].Hunks) != 1 {
		t.Fatalf("hunks=%d", len(files[0].Hunks))
	}
	var adds, dels int
	for _, ln := range files[0].Hunks[0].Lines {
		if ln.Kind == "add" {
			adds++
		}
		if ln.Kind == "del" {
			dels++
		}
	}
	if adds != 2 || dels != 1 {
		t.Fatalf("adds=%d dels=%d", adds, dels)
	}
}
