package secrets

import "testing"

func TestSealOpenAndDigest(t *testing.T) {
	ciphertext, digest, err := Seal("super-secret", "test-key")
	if err != nil {
		t.Fatal(err)
	}
	if ciphertext == "" || ciphertext == "super-secret" {
		t.Fatalf("ciphertext=%q should be non-empty and encrypted", ciphertext)
	}
	if digest != Digest("super-secret") {
		t.Fatalf("digest=%q want %q", digest, Digest("super-secret"))
	}
	opened, err := Open(ciphertext, "test-key")
	if err != nil {
		t.Fatal(err)
	}
	if opened != "super-secret" {
		t.Fatalf("opened=%q", opened)
	}
	if _, err := Open(ciphertext, "wrong-key"); err == nil {
		t.Fatal("Open with wrong key should fail")
	}
}
