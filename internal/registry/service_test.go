package registry_test

import (
	"testing"

	"github.com/ash-repwiki/ash/internal/registry"
	"github.com/ash-repwiki/ash/internal/store"
)

func TestAgentAssetCRUD(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	svc := registry.NewService(db)

	created, err := svc.CreateAgentAsset(registry.CreateAgentAssetRequest{
		SpaceID: "local", Kind: "harness_profile", Name: "default", RefID: "hp_1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.Status != registry.StatusDraft {
		t.Fatalf("status=%s", created.Status)
	}

	list, err := svc.ListAgentAssets("local", "", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Items) != 1 {
		t.Fatalf("items=%d", len(list.Items))
	}

	patched, err := svc.PatchAgentAssetStatus(created.ID, registry.StatusActive)
	if err != nil {
		t.Fatal(err)
	}
	if patched.Status != registry.StatusActive {
		t.Fatalf("status=%s", patched.Status)
	}

	if _, err := svc.CreateAgentAsset(registry.CreateAgentAssetRequest{
		Kind: "nope", Name: "x",
	}); err == nil {
		t.Fatal("expected bad kind")
	}
}

func TestMemoryAssetCRUD(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	svc := registry.NewService(db)

	created, err := svc.CreateMemoryAsset(registry.CreateMemoryAssetRequest{
		Kind: "record", Name: "policy-note", RefID: "mem_1",
	})
	if err != nil {
		t.Fatal(err)
	}
	disabled, err := svc.PatchMemoryAssetStatus(created.ID, registry.StatusDisabled)
	if err != nil {
		t.Fatal(err)
	}
	if disabled.Status != registry.StatusDisabled {
		t.Fatalf("status=%s", disabled.Status)
	}
	list, err := svc.ListMemoryAssets("local", registry.StatusDisabled, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Items) != 1 {
		t.Fatalf("items=%d", len(list.Items))
	}
}
