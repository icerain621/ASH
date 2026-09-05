package rag

import (
	"testing"
)

func TestHashEmbedderDeterministic(t *testing.T) {
	e := NewHashEmbedder(64)
	if e.Dim() != 64 {
		t.Fatalf("Dim()=%d want 64", e.Dim())
	}
	a, err := e.Embed([]string{"hello world"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := e.Embed([]string{"hello world"})
	if err != nil {
		t.Fatal(err)
	}
	if len(a) != 1 || len(a[0]) != 64 {
		t.Fatalf("len=%d/%d want 1/64", len(a), len(a[0]))
	}
	for i := range a[0] {
		if a[0][i] != b[0][i] {
			t.Fatalf("dim %d differs: %v vs %v", i, a[0][i], b[0][i])
		}
	}
}

func TestHashEmbedderDifferentTexts(t *testing.T) {
	e := NewHashEmbedder(64)
	a, err := e.Embed([]string{"alpha"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := e.Embed([]string{"beta"})
	if err != nil {
		t.Fatal(err)
	}
	same := true
	for i := range a[0] {
		if a[0][i] != b[0][i] {
			same = false
			break
		}
	}
	if same {
		t.Fatal("expected different vectors for different texts")
	}
}

func TestHashEmbedderEmptyBatch(t *testing.T) {
	e := NewHashEmbedder(64)
	out, err := e.Embed(nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 0 {
		t.Fatalf("len=%d want 0", len(out))
	}
	out, err = e.Embed([]string{})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 0 {
		t.Fatalf("len=%d want 0", len(out))
	}
}

func TestHashEmbedderEmptyStringErrors(t *testing.T) {
	e := NewHashEmbedder(64)
	_, err := e.Embed([]string{""})
	if err == nil {
		t.Fatal("expected error for empty string")
	}
}
