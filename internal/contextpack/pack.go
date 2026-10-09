package contextpack

import (
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"
)

const (
	MemRetrieve = "mem.retrieve"
	MemInject   = "mem.inject"
)

// LegacyReactMemory is the memory declaration for scenario steps that do not
// name a template yet. It matches tpl.react.
func LegacyReactMemory() []string {
	return []string{MemRetrieve, MemInject}
}

// Hit is one retrieved memory record eligible for injection.
type Hit struct {
	ID    string `json:"id"`
	Title string `json:"title,omitempty"`
}

// SkillCard is the prefix form of a skill. Body is ignored.
type SkillCard struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Body        string `json:"body,omitempty"`
}

// Input is the gathered material for one pack.
type Input struct {
	Issue          string
	RAGRefs        []string
	Memories       []Hit
	Memory         []string
	Skills         []SkillCard
	MaxPrefixRunes int
}

// Pack is the stable context handed to an agent runtime.
type Pack struct {
	Prefix     string   `json:"prefix"`
	Refs       []string `json:"refs"`
	MemoryRefs []string `json:"memoryRefs"`
}

// Build runs Gather → Select → Structure → Compress.
// Memory refs appear only when the declaration includes mem.inject.
func Build(in Input) (Pack, error) {
	if err := ValidateMemoryDecl(in.Memory); err != nil {
		return Pack{}, err
	}
	rag := uniqSorted(in.RAGRefs)
	var hits []Hit
	var memoryRefs []string
	if contains(in.Memory, MemInject) {
		hits = append([]Hit(nil), in.Memories...)
		sort.Slice(hits, func(i, j int) bool { return hits[i].ID < hits[j].ID })
		memoryRefs = make([]string, 0, len(hits))
		for _, hit := range hits {
			if strings.TrimSpace(hit.ID) == "" {
				continue
			}
			memoryRefs = append(memoryRefs, "memory:"+hit.ID)
		}
	}
	refs := make([]string, 0, len(rag)+len(memoryRefs))
	refs = append(refs, rag...)
	refs = append(refs, memoryRefs...)
	prefix := structure(in.Issue, rag, hits, disclosureLines(in.Skills))
	if in.MaxPrefixRunes > 0 {
		prefix = compress(prefix, in.MaxPrefixRunes)
	}
	return Pack{Prefix: prefix, Refs: refs, MemoryRefs: memoryRefs}, nil
}

// ValidateMemoryDecl rejects inject without retrieve.
func ValidateMemoryDecl(decl []string) error {
	if contains(decl, MemInject) && !contains(decl, MemRetrieve) {
		return fmt.Errorf("mem.inject requires mem.retrieve")
	}
	return nil
}

func structure(issue string, rag []string, hits []Hit, skills []string) string {
	var b strings.Builder
	b.WriteString("# issue\n")
	b.WriteString(strings.TrimSpace(issue))
	b.WriteString("\n\n# evidence\n")
	writeRefs(&b, rag)
	b.WriteString("\n# memory\n")
	if len(hits) == 0 {
		b.WriteString("(none)\n")
	} else {
		for _, hit := range hits {
			b.WriteString("- memory:")
			b.WriteString(hit.ID)
			if title := strings.TrimSpace(hit.Title); title != "" {
				b.WriteString(" ")
				b.WriteString(title)
			}
			b.WriteByte('\n')
		}
	}
	if len(skills) > 0 {
		b.WriteString("\n# skills\n")
		for _, line := range skills {
			b.WriteString("- ")
			b.WriteString(line)
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func disclosureLines(cards []SkillCard) []string {
	var out []string
	for _, card := range cards {
		name := strings.TrimSpace(card.Name)
		if name == "" {
			continue
		}
		desc := strings.TrimSpace(card.Description)
		if desc == "" {
			out = append(out, name)
			continue
		}
		out = append(out, name+": "+desc)
	}
	sort.Strings(out)
	return out
}

func writeRefs(b *strings.Builder, refs []string) {
	if len(refs) == 0 {
		b.WriteString("(none)\n")
		return
	}
	for _, ref := range refs {
		b.WriteString("- ")
		b.WriteString(ref)
		b.WriteByte('\n')
	}
}

func compress(prefix string, maxRunes int) string {
	if utf8.RuneCountInString(prefix) <= maxRunes {
		return prefix
	}
	cut := maxRunes
	if cut > 1 {
		cut--
	}
	runes := []rune(prefix)
	return string(runes[:cut]) + "…"
}

func uniqSorted(refs []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(refs))
	for _, ref := range refs {
		ref = strings.TrimSpace(ref)
		if ref == "" {
			continue
		}
		if _, ok := seen[ref]; ok {
			continue
		}
		seen[ref] = struct{}{}
		out = append(out, ref)
	}
	sort.Strings(out)
	return out
}

func contains(list []string, want string) bool {
	for _, item := range list {
		if item == want {
			return true
		}
	}
	return false
}
