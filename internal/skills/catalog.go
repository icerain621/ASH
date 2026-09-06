package skills

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	CatalogSchemaV1      = "ash.skill.catalog.v1"
	DefaultCatalogRelPath = ".ash/skill-catalog.json"
	envCatalogURL         = "ASH_SKILL_CATALOG_URL"
	envCatalogPath        = "ASH_SKILL_CATALOG_PATH"
	catalogFetchTimeout   = 30 * time.Second
	catalogFetchMaxBytes  = 8 << 20
)

// CatalogItem is one org-local pack pointer (filesystem or HTTPS).
type CatalogItem struct {
	Name      string `json:"name"`
	Version   string `json:"version"`
	Publisher string `json:"publisher"`
	URL       string `json:"url"`
	Digest    string `json:"digest,omitempty"`    // expected sha256 of SKILL.md
	Signature string `json:"signature,omitempty"` // pack HMAC hex
}

// Catalog is an org-signed (optional) list of private skill packs. No public marketplace.
type Catalog struct {
	Schema    string        `json:"schema"`
	Signature string        `json:"signature,omitempty"` // optional HMAC over CatalogSignMaterial
	Items     []CatalogItem `json:"items"`
	Source    string        `json:"source,omitempty"` // path or URL used to load
}

// CatalogListResponse is returned by GET /skills/catalog.
type CatalogListResponse struct {
	OK      bool          `json:"ok"`
	Source  string        `json:"source,omitempty"`
	Message string        `json:"message,omitempty"`
	Items   []CatalogItem `json:"items"`
}

// CatalogSignMaterial is the canonical HMAC input for an org catalog.
func CatalogSignMaterial(c Catalog) string {
	items := append([]CatalogItem(nil), c.Items...)
	sort.SliceStable(items, func(i, j int) bool {
		a := items[i].Publisher + "\n" + items[i].Name + "\n" + items[i].Version
		b := items[j].Publisher + "\n" + items[j].Name + "\n" + items[j].Version
		return a < b
	})
	var b strings.Builder
	b.WriteString(firstNonEmpty(strings.TrimSpace(c.Schema), CatalogSchemaV1))
	for _, it := range items {
		b.WriteByte('\n')
		b.WriteString(strings.Join([]string{
			strings.TrimSpace(it.Publisher),
			strings.TrimSpace(it.Name),
			strings.TrimSpace(it.Version),
			normalizeDigest(it.Digest),
			strings.TrimSpace(it.URL),
		}, "\n"))
	}
	return b.String()
}

// SignCatalogHMAC returns lowercase hex HMAC-SHA256 of the catalog material.
func SignCatalogHMAC(key string, c Catalog) string {
	mac := hmac.New(sha256.New, []byte(key))
	_, _ = mac.Write([]byte(CatalogSignMaterial(c)))
	return hex.EncodeToString(mac.Sum(nil))
}

// LoadCatalog resolves ASH_SKILL_CATALOG_URL, ASH_SKILL_CATALOG_PATH, or repoRoot/.ash/skill-catalog.json.
// Missing optional catalog returns empty items with ok semantics (no error).
func LoadCatalog(repoRoot string) (*Catalog, error) {
	if u := strings.TrimSpace(os.Getenv(envCatalogURL)); u != "" {
		raw, err := fetchCatalogBytes(u, "")
		if err != nil {
			return nil, err
		}
		cat, err := parseCatalog(raw, u)
		if err != nil {
			return nil, err
		}
		return cat, nil
	}
	if p := strings.TrimSpace(os.Getenv(envCatalogPath)); p != "" {
		raw, err := os.ReadFile(filepath.Clean(p))
		if err != nil {
			return nil, err
		}
		return parseCatalog(raw, p)
	}
	abs, err := filepath.Abs(firstNonEmpty(strings.TrimSpace(repoRoot), "."))
	if err != nil {
		return nil, err
	}
	path := filepath.Join(abs, filepath.FromSlash(DefaultCatalogRelPath))
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Catalog{Schema: CatalogSchemaV1, Items: nil, Source: path}, nil
		}
		return nil, err
	}
	return parseCatalog(raw, path)
}

func parseCatalog(raw []byte, source string) (*Catalog, error) {
	var cat Catalog
	if err := json.Unmarshal(raw, &cat); err != nil {
		return nil, fmt.Errorf("catalog json: %w", err)
	}
	if cat.Schema != "" && cat.Schema != CatalogSchemaV1 {
		return nil, fmt.Errorf("unsupported catalog schema %q", cat.Schema)
	}
	if cat.Schema == "" {
		cat.Schema = CatalogSchemaV1
	}
	cat.Source = source
	if cat.Items == nil {
		cat.Items = []CatalogItem{}
	}
	return &cat, nil
}

// VerifyCatalogSignature checks optional top-level HMAC; empty signature is allowed (publisher allowlist still applies per item).
func VerifyCatalogSignature(cat *Catalog) error {
	if cat == nil {
		return fmt.Errorf("catalog is nil")
	}
	sig := strings.TrimSpace(cat.Signature)
	if sig == "" {
		return nil
	}
	sig = strings.TrimPrefix(strings.ToLower(sig), "sha256:")
	sig = strings.TrimPrefix(sig, "hmac-sha256:")
	key := PackSigningKey()
	if key == "" {
		return fmt.Errorf("catalog signature present but signing key missing")
	}
	want := SignCatalogHMAC(key, *cat)
	if !hmac.Equal([]byte(want), []byte(sig)) {
		return fmt.Errorf("catalog signature mismatch")
	}
	return nil
}

// ListCatalog loads and optionally verifies the catalog; filters by publisher allowlist.
func ListCatalog(repoRoot string) (*CatalogListResponse, error) {
	cat, err := LoadCatalog(repoRoot)
	if err != nil {
		return &CatalogListResponse{OK: false, Message: err.Error(), Items: []CatalogItem{}}, err
	}
	if err := VerifyCatalogSignature(cat); err != nil {
		return &CatalogListResponse{OK: false, Source: cat.Source, Message: err.Error(), Items: []CatalogItem{}}, err
	}
	items := make([]CatalogItem, 0, len(cat.Items))
	for _, it := range cat.Items {
		if !publisherAllowed(it.Publisher) {
			continue
		}
		items = append(items, it)
	}
	return &CatalogListResponse{
		OK: true, Source: cat.Source, Items: items,
		Message: fmt.Sprintf("%d item(s)", len(items)),
	}, nil
}

// InstallFromCatalog fetches the pack URL for name(/version) and installs via InstallPackBytes.
func InstallFromCatalog(repoRoot, spaceID, name, version string) (*PackInstallResult, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("name required")
	}
	cat, err := LoadCatalog(repoRoot)
	if err != nil {
		return nil, err
	}
	if err := VerifyCatalogSignature(cat); err != nil {
		return nil, err
	}
	version = strings.TrimSpace(version)
	var matched *CatalogItem
	for i := range cat.Items {
		it := &cat.Items[i]
		if strings.TrimSpace(it.Name) != name {
			continue
		}
		if version != "" && strings.TrimSpace(it.Version) != version {
			continue
		}
		matched = it
		break
	}
	if matched == nil {
		if version != "" {
			return nil, fmt.Errorf("catalog entry not found: %s@%s", name, version)
		}
		return nil, fmt.Errorf("catalog entry not found: %s", name)
	}
	if !publisherAllowed(matched.Publisher) {
		return nil, fmt.Errorf("publisher %q not in ASH_SKILL_PACK_ALLOWLIST", matched.Publisher)
	}
	baseDir := ""
	if cat.Source != "" && !strings.Contains(cat.Source, "://") {
		baseDir = filepath.Dir(cat.Source)
	}
	zipBytes, err := fetchCatalogBytes(matched.URL, baseDir)
	if err != nil {
		return nil, fmt.Errorf("fetch pack: %w", err)
	}
	if dig := strings.TrimSpace(matched.Digest); dig != "" {
		if err := assertPackDigest(zipBytes, dig); err != nil {
			return nil, err
		}
	}
	return InstallPackBytes(repoRoot, spaceID, zipBytes, matched.Signature)
}

func assertPackDigest(zipBytes []byte, wantDigest string) error {
	files, err := readZipFiles(zipBytes)
	if err != nil {
		return err
	}
	skillRaw, ok := files[PackSkillFile]
	if !ok {
		return fmt.Errorf("missing %s in pack", PackSkillFile)
	}
	got := digestSKILL(skillRaw)
	if normalizeDigest(got) != normalizeDigest(wantDigest) {
		return fmt.Errorf("catalog digest mismatch")
	}
	return nil
}

func fetchCatalogBytes(rawURL, baseDir string) ([]byte, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return nil, fmt.Errorf("empty url")
	}
	lower := strings.ToLower(rawURL)
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
		client := &http.Client{Timeout: catalogFetchTimeout}
		resp, err := client.Get(rawURL)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, fmt.Errorf("http %s", resp.Status)
		}
		return io.ReadAll(io.LimitReader(resp.Body, catalogFetchMaxBytes))
	}
	if strings.HasPrefix(lower, "file://") {
		u, err := url.Parse(rawURL)
		if err != nil {
			return nil, err
		}
		p := u.Path
		if runtimeIsWindows() && strings.HasPrefix(p, "/") && len(p) >= 3 && p[2] == ':' {
			p = p[1:]
		}
		return os.ReadFile(filepath.Clean(p))
	}
	path := rawURL
	if !filepath.IsAbs(path) && baseDir != "" {
		path = filepath.Join(baseDir, path)
	}
	return os.ReadFile(filepath.Clean(path))
}

func runtimeIsWindows() bool {
	return os.PathSeparator == '\\'
}
