package skills

import (
	"archive/zip"
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	PackSchemaV1     = "ash.skill.pack.v1"
	PackManifestName = "ash-skill-pack.json"
	PackSkillFile    = "SKILL.md"
)

// PackManifest describes a private skill pack (no marketplace registry).
type PackManifest struct {
	Schema    string `json:"schema"`
	Name      string `json:"name"`
	Version   string `json:"version"`
	Publisher string `json:"publisher"`
	Digest    string `json:"digest"` // sha256:<hex> of SKILL.md
	Signature string `json:"signature,omitempty"`
}

// PackVerifyResult is returned by VerifyPack / dry-run API.
type PackVerifyResult struct {
	OK        bool   `json:"ok"`
	Name      string `json:"name,omitempty"`
	Version   string `json:"version,omitempty"`
	Publisher string `json:"publisher,omitempty"`
	Digest    string `json:"digest,omitempty"`
	Message   string `json:"message,omitempty"`
}

// PackInstallResult is returned after a successful install.
type PackInstallResult struct {
	OK        bool   `json:"ok"`
	Name      string `json:"name"`
	Version   string `json:"version"`
	Publisher string `json:"publisher"`
	Path      string `json:"path"`
	RepoRoot  string `json:"repoRoot"`
}

// PackSignMaterial is the canonical HMAC input: publisher\nname\nversion\ndigestHex
func PackSignMaterial(publisher, name, version, digestHex string) string {
	return strings.Join([]string{
		strings.TrimSpace(publisher),
		strings.TrimSpace(name),
		strings.TrimSpace(version),
		strings.TrimPrefix(strings.TrimSpace(digestHex), "sha256:"),
	}, "\n")
}

// SignPackHMAC returns lowercase hex HMAC-SHA256.
func SignPackHMAC(key, publisher, name, version, digestHex string) string {
	mac := hmac.New(sha256.New, []byte(key))
	_, _ = mac.Write([]byte(PackSignMaterial(publisher, name, version, digestHex)))
	return hex.EncodeToString(mac.Sum(nil))
}

// PackSigningKey returns ASH_SKILL_PACK_SIGNING_KEY, falling back to ASH_PLUGIN_SIGNING_KEY.
func PackSigningKey() string {
	if k := strings.TrimSpace(os.Getenv("ASH_SKILL_PACK_SIGNING_KEY")); k != "" {
		return k
	}
	return strings.TrimSpace(os.Getenv("ASH_PLUGIN_SIGNING_KEY"))
}

func normalizeDigest(d string) string {
	d = strings.TrimSpace(strings.ToLower(d))
	d = strings.TrimPrefix(d, "sha256:")
	return d
}

func digestSKILL(content []byte) string {
	sum := sha256.Sum256(content)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// BuildPackZip creates a zip from a skill directory containing SKILL.md.
// Manifest signature is left empty (caller signs).
func BuildPackZip(skillDir, publisher, version string) ([]byte, PackManifest, error) {
	skillDir = strings.TrimSpace(skillDir)
	skillPath := filepath.Join(skillDir, PackSkillFile)
	raw, err := os.ReadFile(skillPath)
	if err != nil {
		return nil, PackManifest{}, fmt.Errorf("read SKILL.md: %w", err)
	}
	sk, err := ParseFile(skillPath, skillDir)
	if err != nil {
		return nil, PackManifest{}, err
	}
	pub := strings.TrimSpace(publisher)
	if pub == "" {
		pub = "local"
	}
	ver := strings.TrimSpace(version)
	if ver == "" {
		ver = "0.0.0"
	}
	man := PackManifest{
		Schema:    PackSchemaV1,
		Name:      sk.Name,
		Version:   ver,
		Publisher: pub,
		Digest:    digestSKILL(raw),
	}
	manJSON, err := json.MarshalIndent(man, "", "  ")
	if err != nil {
		return nil, PackManifest{}, err
	}

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	if err := writeZipFile(zw, PackManifestName, manJSON); err != nil {
		_ = zw.Close()
		return nil, PackManifest{}, err
	}
	if err := writeZipFile(zw, PackSkillFile, raw); err != nil {
		_ = zw.Close()
		return nil, PackManifest{}, err
	}
	// Optional sibling files (no nested dirs for MVP).
	entries, _ := os.ReadDir(skillDir)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if name == PackSkillFile || name == PackManifestName {
			continue
		}
		b, err := os.ReadFile(filepath.Join(skillDir, name))
		if err != nil {
			continue
		}
		if err := writeZipFile(zw, name, b); err != nil {
			_ = zw.Close()
			return nil, PackManifest{}, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, PackManifest{}, err
	}
	return buf.Bytes(), man, nil
}

func writeZipFile(zw *zip.Writer, name string, body []byte) error {
	w, err := zw.Create(name)
	if err != nil {
		return err
	}
	_, err = w.Write(body)
	return err
}

// VerifyPackBytes validates zip contents + HMAC signature (fail closed without key).
func VerifyPackBytes(zipBytes []byte, signatureOverride string) (*PackVerifyResult, *PackManifest, map[string][]byte, error) {
	files, err := readZipFiles(zipBytes)
	if err != nil {
		return &PackVerifyResult{OK: false, Message: err.Error()}, nil, nil, err
	}
	manRaw, ok := files[PackManifestName]
	if !ok {
		err := fmt.Errorf("missing %s", PackManifestName)
		return &PackVerifyResult{OK: false, Message: err.Error()}, nil, nil, err
	}
	var man PackManifest
	if err := json.Unmarshal(manRaw, &man); err != nil {
		return &PackVerifyResult{OK: false, Message: "manifest json: " + err.Error()}, nil, nil, err
	}
	if man.Schema != "" && man.Schema != PackSchemaV1 {
		err := fmt.Errorf("unsupported schema %q", man.Schema)
		return &PackVerifyResult{OK: false, Message: err.Error()}, &man, nil, err
	}
	skillRaw, ok := files[PackSkillFile]
	if !ok {
		err := fmt.Errorf("missing %s", PackSkillFile)
		return &PackVerifyResult{OK: false, Message: err.Error()}, &man, nil, err
	}
	if _, err := ParseFileContent(skillRaw, PackSkillFile); err != nil {
		return &PackVerifyResult{OK: false, Message: "SKILL.md: " + err.Error()}, &man, nil, err
	}
	wantDigest := digestSKILL(skillRaw)
	if normalizeDigest(man.Digest) != normalizeDigest(wantDigest) {
		err := fmt.Errorf("digest mismatch")
		return &PackVerifyResult{OK: false, Message: err.Error(), Name: man.Name, Digest: wantDigest}, &man, nil, err
	}
	man.Digest = wantDigest

	sig := strings.TrimSpace(signatureOverride)
	if sig == "" {
		sig = man.Signature
	}
	sig = strings.TrimPrefix(strings.ToLower(sig), "sha256:")
	sig = strings.TrimPrefix(sig, "hmac-sha256:")

	key := PackSigningKey()
	if key == "" {
		err := fmt.Errorf("skill pack signing key required (ASH_SKILL_PACK_SIGNING_KEY or ASH_PLUGIN_SIGNING_KEY)")
		return &PackVerifyResult{OK: false, Message: err.Error(), Name: man.Name, Version: man.Version, Publisher: man.Publisher, Digest: man.Digest}, &man, nil, err
	}
	if sig == "" {
		err := fmt.Errorf("signature required")
		return &PackVerifyResult{OK: false, Message: err.Error(), Name: man.Name, Version: man.Version, Publisher: man.Publisher, Digest: man.Digest}, &man, nil, err
	}
	want := SignPackHMAC(key, man.Publisher, man.Name, man.Version, man.Digest)
	if !hmac.Equal([]byte(want), []byte(sig)) {
		err := fmt.Errorf("signature mismatch")
		return &PackVerifyResult{OK: false, Message: err.Error(), Name: man.Name, Version: man.Version, Publisher: man.Publisher, Digest: man.Digest}, &man, nil, err
	}
	if !publisherAllowed(man.Publisher) {
		err := fmt.Errorf("publisher %q not in ASH_SKILL_PACK_ALLOWLIST", man.Publisher)
		return &PackVerifyResult{OK: false, Message: err.Error(), Name: man.Name, Version: man.Version, Publisher: man.Publisher, Digest: man.Digest}, &man, nil, err
	}
	return &PackVerifyResult{
		OK: true, Name: man.Name, Version: man.Version, Publisher: man.Publisher, Digest: man.Digest,
		Message: "signature ok",
	}, &man, files, nil
}

func publisherAllowed(publisher string) bool {
	list := strings.TrimSpace(os.Getenv("ASH_SKILL_PACK_ALLOWLIST"))
	if list == "" || list == "*" {
		return true
	}
	want := strings.TrimSpace(publisher)
	for _, p := range strings.Split(list, ",") {
		if strings.TrimSpace(p) == want {
			return true
		}
	}
	return false
}

func spaceAllowed(spaceID string) bool {
	list := strings.TrimSpace(os.Getenv("ASH_SKILL_PACK_SPACES"))
	if list == "" || list == "*" {
		return true
	}
	want := strings.TrimSpace(spaceID)
	if want == "" {
		want = "local"
	}
	for _, s := range strings.Split(list, ",") {
		if strings.TrimSpace(s) == want {
			return true
		}
	}
	return false
}

func readZipFiles(zipBytes []byte) (map[string][]byte, error) {
	zr, err := zip.NewReader(bytes.NewReader(zipBytes), int64(len(zipBytes)))
	if err != nil {
		return nil, fmt.Errorf("zip: %w", err)
	}
	out := map[string][]byte{}
	for _, f := range zr.File {
		name := filepath.ToSlash(f.Name)
		if strings.Contains(name, "..") || strings.HasPrefix(name, "/") {
			return nil, fmt.Errorf("unsafe zip path %q", f.Name)
		}
		if f.FileInfo().IsDir() {
			continue
		}
		// MVP: flat files only
		if strings.Contains(name, "/") {
			return nil, fmt.Errorf("nested paths not supported: %s", name)
		}
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		b, err := io.ReadAll(io.LimitReader(rc, 2<<20))
		_ = rc.Close()
		if err != nil {
			return nil, err
		}
		out[name] = b
	}
	return out, nil
}

// InstallPackBytes verifies then extracts into .ash/skills/<name>/.
func InstallPackBytes(repoRoot, spaceID string, zipBytes []byte, signatureOverride string) (*PackInstallResult, error) {
	spaceID = firstNonEmpty(strings.TrimSpace(spaceID), "local")
	if !spaceAllowed(spaceID) {
		return nil, fmt.Errorf("space %q not in ASH_SKILL_PACK_SPACES", spaceID)
	}
	vr, man, files, err := VerifyPackBytes(zipBytes, signatureOverride)
	if err != nil || vr == nil || !vr.OK || man == nil {
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("verify failed: %s", vr.Message)
	}
	abs, err := filepath.Abs(firstNonEmpty(strings.TrimSpace(repoRoot), "."))
	if err != nil {
		return nil, err
	}
	dest := filepath.Join(abs, ".ash", "skills", man.Name)
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return nil, err
	}
	names := make([]string, 0, len(files))
	for name := range files {
		if name == PackManifestName {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if err := os.WriteFile(filepath.Join(dest, name), files[name], 0o644); err != nil {
			return nil, err
		}
	}
	// Persist signed manifest for audit (filesystem ledger; no SQL).
	man.Signature = strings.TrimSpace(signatureOverride)
	if man.Signature == "" {
		// keep embedded if any
	}
	manJSON, _ := json.MarshalIndent(man, "", "  ")
	_ = os.WriteFile(filepath.Join(dest, PackManifestName), manJSON, 0o644)

	return &PackInstallResult{
		OK: true, Name: man.Name, Version: man.Version, Publisher: man.Publisher,
		Path: filepath.ToSlash(filepath.Join(".ash", "skills", man.Name)),
		RepoRoot: abs,
	}, nil
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// ParseFileContent parses SKILL.md bytes (for pack verify without a real path).
func ParseFileContent(raw []byte, fakePath string) (*Skill, error) {
	fm, body, err := splitFrontmatter(string(raw))
	if err != nil {
		return nil, err
	}
	var meta frontmatter
	if err := yaml.Unmarshal([]byte(fm), &meta); err != nil {
		return nil, fmt.Errorf("frontmatter: %w", err)
	}
	name := strings.TrimSpace(meta.Name)
	if name == "" {
		name = "unnamed"
	}
	if meta.Description == "" {
		return nil, fmt.Errorf("description required")
	}
	return &Skill{
		ID: name, Name: name, Description: strings.TrimSpace(meta.Description),
		License: strings.TrimSpace(meta.License), Path: fakePath,
		Body: strings.TrimSpace(body), ContextRef: "skill:" + name,
	}, nil
}
