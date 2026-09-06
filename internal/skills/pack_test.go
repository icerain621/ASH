package skills

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildSignVerifyInstallPack(t *testing.T) {
	t.Setenv("ASH_SKILL_PACK_SIGNING_KEY", "dx22-test-key")
	t.Setenv("ASH_SKILL_PACK_ALLOWLIST", "acme")
	t.Setenv("ASH_SKILL_PACK_SPACES", "local")

	src := t.TempDir()
	skillMD := "---\nname: demo-pack\ndescription: DX22 fixture skill\n---\n\n# Demo\n"
	if err := os.WriteFile(filepath.Join(src, "SKILL.md"), []byte(skillMD), 0o644); err != nil {
		t.Fatal(err)
	}

	zipBytes, man, err := BuildPackZip(src, "acme", "1.2.3")
	if err != nil {
		t.Fatal(err)
	}
	sig := SignPackHMAC(PackSigningKey(), man.Publisher, man.Name, man.Version, man.Digest)

	vr, _, _, err := VerifyPackBytes(zipBytes, sig)
	if err != nil || vr == nil || !vr.OK {
		t.Fatalf("verify: vr=%+v err=%v", vr, err)
	}

	repo := t.TempDir()
	inst, err := InstallPackBytes(repo, "local", zipBytes, sig)
	if err != nil {
		t.Fatal(err)
	}
	if !inst.OK || inst.Name != "demo-pack" {
		t.Fatalf("install=%+v", inst)
	}
	if _, err := os.Stat(filepath.Join(repo, ".ash", "skills", "demo-pack", "SKILL.md")); err != nil {
		t.Fatal(err)
	}
	list, err := ScanRepo(repo)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, sk := range list.Items {
		if sk.ID == "demo-pack" {
			found = true
		}
	}
	if !found {
		t.Fatalf("scan missing demo-pack: %+v", list.Items)
	}
}

func TestVerifyPackRejectsBadSignature(t *testing.T) {
	t.Setenv("ASH_SKILL_PACK_SIGNING_KEY", "dx22-test-key")
	t.Setenv("ASH_SKILL_PACK_ALLOWLIST", "*")

	src := t.TempDir()
	if err := os.WriteFile(filepath.Join(src, "SKILL.md"), []byte("---\nname: x\ndescription: y\n---\n\nbody\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	zipBytes, _, err := BuildPackZip(src, "local", "0.1.0")
	if err != nil {
		t.Fatal(err)
	}
	vr, _, _, err := VerifyPackBytes(zipBytes, "deadbeef")
	if err == nil || (vr != nil && vr.OK) {
		t.Fatalf("expected signature failure, vr=%+v err=%v", vr, err)
	}
}

func TestVerifyPackRejectsUnknownPublisher(t *testing.T) {
	t.Setenv("ASH_SKILL_PACK_SIGNING_KEY", "dx22-test-key")
	t.Setenv("ASH_SKILL_PACK_ALLOWLIST", "only-acme")

	src := t.TempDir()
	if err := os.WriteFile(filepath.Join(src, "SKILL.md"), []byte("---\nname: x\ndescription: y\n---\n\nbody\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	zipBytes, man, err := BuildPackZip(src, "other", "0.1.0")
	if err != nil {
		t.Fatal(err)
	}
	sig := SignPackHMAC(PackSigningKey(), man.Publisher, man.Name, man.Version, man.Digest)
	vr, _, _, err := VerifyPackBytes(zipBytes, sig)
	if err == nil || (vr != nil && vr.OK) {
		t.Fatalf("expected allowlist failure, vr=%+v err=%v", vr, err)
	}
}

func TestInstallPackRejectsDisallowedSpace(t *testing.T) {
	t.Setenv("ASH_SKILL_PACK_SIGNING_KEY", "dx22-test-key")
	t.Setenv("ASH_SKILL_PACK_ALLOWLIST", "*")
	t.Setenv("ASH_SKILL_PACK_SPACES", "prod")

	src := t.TempDir()
	if err := os.WriteFile(filepath.Join(src, "SKILL.md"), []byte("---\nname: x\ndescription: y\n---\n\nbody\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	zipBytes, man, err := BuildPackZip(src, "local", "0.1.0")
	if err != nil {
		t.Fatal(err)
	}
	sig := SignPackHMAC(PackSigningKey(), man.Publisher, man.Name, man.Version, man.Digest)
	_, err = InstallPackBytes(t.TempDir(), "local", zipBytes, sig)
	if err == nil {
		t.Fatal("expected space deny")
	}
}
