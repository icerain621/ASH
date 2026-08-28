package diffparse

import (
	"strconv"
	"strings"
)

// FileDiff is one file section of a unified diff.
type FileDiff struct {
	Path  string `json:"path"`
	Hunks []Hunk `json:"hunks"`
}

// Hunk is a contiguous change block.
type Hunk struct {
	Header string     `json:"header"`
	OldStart int      `json:"oldStart"`
	NewStart int      `json:"newStart"`
	Lines  []DiffLine `json:"lines"`
}

// DiffLine is a single annotated line within a hunk.
type DiffLine struct {
	Kind    string `json:"kind"` // context|add|del|meta
	Text    string `json:"text"`
	OldNo   *int   `json:"oldNo,omitempty"`
	NewNo   *int   `json:"newNo,omitempty"`
	Index   int    `json:"index"` // stable index within file for anchoring
}

// ParseUnified parses a unified diff into file/hunk/line structures.
func ParseUnified(raw string) []FileDiff {
	raw = strings.ReplaceAll(strings.ReplaceAll(raw, "\r\n", "\n"), "\r", "\n")
	lines := strings.Split(raw, "\n")
	var files []FileDiff
	var cur *FileDiff
	var hunk *Hunk
	fileLineIdx := 0
	oldNo, newNo := 0, 0

	flushHunk := func() {
		if cur != nil && hunk != nil && len(hunk.Lines) > 0 {
			cur.Hunks = append(cur.Hunks, *hunk)
		}
		hunk = nil
	}
	flushFile := func() {
		flushHunk()
		if cur != nil && (cur.Path != "" || len(cur.Hunks) > 0) {
			files = append(files, *cur)
		}
		cur = nil
		fileLineIdx = 0
	}

	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "diff --git "):
			flushFile()
			path := parseGitPath(line)
			cur = &FileDiff{Path: path}
		case strings.HasPrefix(line, "+++ "):
			if cur == nil {
				cur = &FileDiff{}
			}
			p := strings.TrimPrefix(line, "+++ ")
			p = strings.TrimPrefix(p, "b/")
			if p != "/dev/null" && p != "" {
				cur.Path = p
			}
		case strings.HasPrefix(line, "--- "):
			// keep for path if +++ missing
			if cur == nil {
				cur = &FileDiff{}
			}
			p := strings.TrimPrefix(line, "--- ")
			p = strings.TrimPrefix(p, "a/")
			if cur.Path == "" && p != "/dev/null" {
				cur.Path = p
			}
		case strings.HasPrefix(line, "@@"):
			flushHunk()
			if cur == nil {
				cur = &FileDiff{Path: "unknown"}
			}
			os, ns := parseHunkHeader(line)
			oldNo, newNo = os, ns
			hunk = &Hunk{Header: line, OldStart: os, NewStart: ns}
		default:
			if hunk == nil {
				continue
			}
			dl := DiffLine{Text: line, Index: fileLineIdx}
			fileLineIdx++
			switch {
			case strings.HasPrefix(line, "+"):
				dl.Kind = "add"
				n := newNo
				dl.NewNo = &n
				newNo++
			case strings.HasPrefix(line, "-"):
				dl.Kind = "del"
				o := oldNo
				dl.OldNo = &o
				oldNo++
			case strings.HasPrefix(line, "\\"):
				dl.Kind = "meta"
			default:
				dl.Kind = "context"
				o, n := oldNo, newNo
				dl.OldNo = &o
				dl.NewNo = &n
				oldNo++
				newNo++
			}
			hunk.Lines = append(hunk.Lines, dl)
		}
	}
	flushFile()
	return files
}

func parseGitPath(line string) string {
	// diff --git a/foo b/foo
	parts := strings.Fields(line)
	if len(parts) >= 4 {
		p := parts[3]
		return strings.TrimPrefix(p, "b/")
	}
	return "unknown"
}

func parseHunkHeader(header string) (oldStart, newStart int) {
	// @@ -12,5 +14,7 @@
	oldStart, newStart = 1, 1
	i := strings.Index(header, "-")
	j := strings.Index(header, "+")
	if i < 0 || j < 0 {
		return
	}
	oldPart := header[i+1 : j]
	newPart := header[j+1:]
	if k := strings.IndexAny(newPart, " @"); k >= 0 {
		newPart = newPart[:k]
	}
	oldStart = atoiFirst(oldPart)
	newStart = atoiFirst(newPart)
	return
}

func atoiFirst(s string) int {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, ','); i >= 0 {
		s = s[:i]
	}
	n, _ := strconv.Atoi(strings.TrimSpace(s))
	if n <= 0 {
		return 1
	}
	return n
}
