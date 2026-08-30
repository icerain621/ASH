package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Skill is a parsed SKILL.md (Agent Skills frontmatter subset).
type Skill struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	License     string `json:"license,omitempty"`
	Path        string `json:"path"`
	RelPath     string `json:"relPath"`
	Body        string `json:"body,omitempty"`
	ContextRef  string `json:"contextRef"`
}

type ListResponse struct {
	Items    []Skill `json:"items"`
	RepoRoot string  `json:"repoRoot"`
}

type frontmatter struct {
	Name          string `yaml:"name"`
	Description   string `yaml:"description"`
	License       string `yaml:"license"`
	Compatibility string `yaml:"compatibility"`
}

// ScanRepo finds SKILL.md under .ash/skills/*/ and skills/*/.
func ScanRepo(repoRoot string) (*ListResponse, error) {
	root := strings.TrimSpace(repoRoot)
	if root == "" {
		root = "."
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	seen := map[string]Skill{}
	for _, base := range []string{
		filepath.Join(abs, ".ash", "skills"),
		filepath.Join(abs, "skills"),
	} {
		entries, err := os.ReadDir(base)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			path := filepath.Join(base, e.Name(), "SKILL.md")
			sk, err := ParseFile(path, abs)
			if err != nil {
				continue
			}
			seen[sk.ID] = *sk
		}
	}
	items := make([]Skill, 0, len(seen))
	for _, sk := range seen {
		items = append(items, sk)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return &ListResponse{Items: items, RepoRoot: abs}, nil
}

// Get loads one skill by id (name).
func Get(repoRoot, skillID string) (*Skill, error) {
	skillID = strings.TrimSpace(skillID)
	if skillID == "" {
		return nil, fmt.Errorf("skillId required")
	}
	list, err := ScanRepo(repoRoot)
	if err != nil {
		return nil, err
	}
	for i := range list.Items {
		if list.Items[i].ID == skillID || list.Items[i].Name == skillID {
			sk := list.Items[i]
			// re-read with body
			full, err := ParseFile(sk.Path, list.RepoRoot)
			if err != nil {
				return nil, err
			}
			return full, nil
		}
	}
	return nil, fmt.Errorf("skill %q not found", skillID)
}

// ContextRefsForWanted returns skill: refs for declared ids that exist in repo.
func ContextRefsForWanted(repoRoot string, wanted []string) []string {
	if len(wanted) == 0 {
		return nil
	}
	list, err := ScanRepo(repoRoot)
	if err != nil || list == nil {
		return nil
	}
	byID := map[string]Skill{}
	for _, sk := range list.Items {
		byID[sk.ID] = sk
	}
	var refs []string
	seen := map[string]struct{}{}
	for _, id := range wanted {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		sk, ok := byID[id]
		if !ok {
			continue
		}
		if _, dup := seen[sk.ContextRef]; dup {
			continue
		}
		seen[sk.ContextRef] = struct{}{}
		refs = append(refs, sk.ContextRef)
	}
	return refs
}

// ParseFile parses a SKILL.md path.
func ParseFile(path, repoAbs string) (*Skill, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
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
		name = filepath.Base(filepath.Dir(path))
	}
	if meta.Description == "" {
		return nil, fmt.Errorf("description required in %s", path)
	}
	rel := path
	if repoAbs != "" {
		if r, err := filepath.Rel(repoAbs, path); err == nil {
			rel = filepath.ToSlash(r)
		}
	}
	return &Skill{
		ID: name, Name: name, Description: strings.TrimSpace(meta.Description),
		License: strings.TrimSpace(meta.License), Path: path, RelPath: rel,
		Body: strings.TrimSpace(body), ContextRef: "skill:" + name,
	}, nil
}

func splitFrontmatter(raw string) (fm, body string, err error) {
	s := strings.TrimPrefix(raw, "\uFEFF")
	s = strings.TrimLeft(s, "\r\n")
	if !strings.HasPrefix(s, "---") {
		return "", "", fmt.Errorf("missing YAML frontmatter")
	}
	rest := s[3:]
	rest = strings.TrimPrefix(rest, "\r\n")
	rest = strings.TrimPrefix(rest, "\n")
	idx := strings.Index(rest, "\n---")
	if idx < 0 {
		return "", "", fmt.Errorf("unclosed frontmatter")
	}
	fm = rest[:idx]
	body = rest[idx+4:]
	body = strings.TrimPrefix(body, "\r\n")
	body = strings.TrimPrefix(body, "\n")
	return fm, body, nil
}
