package skills

import "strings"

// DisclosureLine is the prefix form of a skill: name and description, never the body.
func DisclosureLine(sk Skill) string {
	name := strings.TrimSpace(sk.Name)
	if name == "" {
		name = strings.TrimSpace(sk.ID)
	}
	if name == "" {
		return ""
	}
	desc := strings.TrimSpace(sk.Description)
	if desc == "" {
		return name
	}
	return name + ": " + desc
}

// DisclosureLines lists skills for a context prefix. Bodies stay out.
func DisclosureLines(items []Skill) []string {
	var out []string
	for _, sk := range items {
		line := DisclosureLine(sk)
		if line == "" {
			continue
		}
		out = append(out, line)
	}
	return out
}
