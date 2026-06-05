package security

import (
	"regexp"
	"strings"
)

type LeakFinding struct {
	Source  string `json:"source"`
	Ref     string `json:"ref"`
	Pattern string `json:"pattern"`
	Snippet string `json:"snippet"`
}

var leakPatterns = []*regexp.Regexp{
	// JSON "key":"value"
	regexp.MustCompile(`(?i)"(?:api[_-]?key|secret|password|token|authorization)"\s*:\s*"([^"]{8,})"`),
	// key=value / key: value (non-JSON)
	regexp.MustCompile(`(?i)(api[_-]?key|secret|password|token|authorization)\s*[:=]\s*["']?([^\s"',}{]{8,})`),
	regexp.MustCompile(`(?i)bearer\s+([a-zA-Z0-9_\-\.]{20,})`),
	regexp.MustCompile(`sk-[a-zA-Z0-9]{20,}`),
	regexp.MustCompile(`(?i)-----BEGIN (?:RSA |EC )?PRIVATE KEY-----`),
}

var redactPatterns = []*regexp.Regexp{
	// JSON "key":"value"
	regexp.MustCompile(`(?i)("(?:api[_-]?key|secret|password|token|authorization)"\s*:\s*")([^"]{4,})"`),
	// key=value / key: value (non-JSON)
	regexp.MustCompile(`(?i)((?:api[_-]?key|secret|password|token|authorization)\s*[:=]\s*["']?)([^\s"',}{]{4,})`),
	regexp.MustCompile(`(?i)(bearer\s+)([a-zA-Z0-9_\-\.]{8,})`),
	regexp.MustCompile(`(sk-[a-zA-Z0-9]{8,})`),
}

// FindLeaks returns pattern matches that look like plaintext secrets.
func FindLeaks(source, ref, text string) []LeakFinding {
	if strings.TrimSpace(text) == "" {
		return nil
	}
	var out []LeakFinding
	for _, pattern := range leakPatterns {
		for _, match := range pattern.FindAllStringSubmatch(text, 8) {
			snippet := match[0]
			if len(snippet) > 120 {
				snippet = snippet[:117] + "..."
			}
			out = append(out, LeakFinding{
				Source: source, Ref: ref, Pattern: pattern.String(), Snippet: snippet,
			})
		}
	}
	return out
}

// RedactJSON masks likely secret values inside JSON/text payloads.
func RedactJSON(raw string) string {
	if raw == "" {
		return raw
	}
	out := raw
	for _, pattern := range redactPatterns {
		out = pattern.ReplaceAllString(out, `${1}***REDACTED***`)
	}
	return out
}

// ScanTexts scans multiple text blobs.
func ScanTexts(items []struct{ Source, Ref, Text string }) []LeakFinding {
	var out []LeakFinding
	for _, item := range items {
		out = append(out, FindLeaks(item.Source, item.Ref, item.Text)...)
	}
	return out
}
