package rag

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// TreeSitterIndexer extracts symbols via pure-Go parsers (go/ast + structured
// TS/JS/YAML). Named for the SymbolIndexer slot; avoids CGO tree-sitter so
// Windows CI stays green. Unsupported languages return err so rebuild can
// fall back to regex per file.
type TreeSitterIndexer struct{}

func NewTreeSitterIndexer() TreeSitterIndexer { return TreeSitterIndexer{} }

func (TreeSitterIndexer) Name() string { return "treesitter" }

func (TreeSitterIndexer) Available() bool { return true }

func (t TreeSitterIndexer) IndexFile(path string, content []byte) ([]SymbolHit, error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".go":
		return extractGoSymbols(path, content)
	case ".ts", ".tsx", ".js", ".jsx":
		return extractJSTSSymbols(content)
	case ".yaml", ".yml":
		return extractYAMLSymbols(content)
	default:
		return nil, fmt.Errorf("treesitter: unsupported language %s", ext)
	}
}

func extractGoSymbols(path string, content []byte) ([]SymbolHit, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, content, parser.SkipObjectResolution)
	if err != nil {
		return nil, fmt.Errorf("treesitter go parse: %w", err)
	}
	var hits []SymbolHit
	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if d.Name == nil || d.Name.Name == "" {
				continue
			}
			kind := "func"
			if d.Recv != nil {
				kind = "method"
			}
			hits = append(hits, SymbolHit{
				Name: d.Name.Name,
				Kind: kind,
				Line: fset.Position(d.Pos()).Line,
			})
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					if s.Name == nil || s.Name.Name == "" {
						continue
					}
					hits = append(hits, SymbolHit{
						Name: s.Name.Name,
						Kind: "type",
						Line: fset.Position(s.Pos()).Line,
					})
				case *ast.ValueSpec:
					kind := "var"
					if d.Tok.String() == "const" {
						kind = "const"
					}
					for _, name := range s.Names {
						if name == nil || name.Name == "" || name.Name == "_" {
							continue
						}
						hits = append(hits, SymbolHit{
							Name: name.Name,
							Kind: kind,
							Line: fset.Position(name.Pos()).Line,
						})
					}
				}
			}
		}
	}
	return hits, nil
}

var (
	jsFuncPattern      = regexp.MustCompile(`(?m)^\s*(?:export\s+)?(?:async\s+)?function\s+\*?\s*([A-Za-z_$][\w$]*)`)
	jsClassPattern     = regexp.MustCompile(`(?m)^\s*(?:export\s+)?(?:abstract\s+)?class\s+([A-Za-z_$][\w$]*)`)
	jsInterfacePattern = regexp.MustCompile(`(?m)^\s*(?:export\s+)?interface\s+([A-Za-z_$][\w$]*)`)
	jsTypePattern      = regexp.MustCompile(`(?m)^\s*(?:export\s+)?type\s+([A-Za-z_$][\w$]*)\s*=`)
	jsConstPattern     = regexp.MustCompile(`(?m)^\s*(?:export\s+)?(?:const|let|var)\s+([A-Za-z_$][\w$]*)\s*=`)
	jsMethodPattern    = regexp.MustCompile(`(?m)^\s+(?:async\s+)?(?:get\s+|set\s+)?([A-Za-z_$][\w$]*)\s*\(`)
)

func extractJSTSSymbols(content []byte) ([]SymbolHit, error) {
	text := string(content)
	lines := splitLines(text)
	seen := map[string]bool{}
	var hits []SymbolHit
	add := func(name, kind string, line int) {
		key := name + "\x00" + kind + "\x00" + fmt.Sprintf("%d", line)
		if name == "" || seen[key] {
			return
		}
		seen[key] = true
		hits = append(hits, SymbolHit{Name: name, Kind: kind, Line: line})
	}
	for _, reKind := range []struct {
		re   *regexp.Regexp
		kind string
	}{
		{jsFuncPattern, "func"},
		{jsClassPattern, "class"},
		{jsInterfacePattern, "interface"},
		{jsTypePattern, "type"},
		{jsConstPattern, "var"},
	} {
		for _, m := range reKind.re.FindAllStringSubmatchIndex(text, -1) {
			if len(m) < 4 {
				continue
			}
			name := text[m[2]:m[3]]
			line := 1 + strings.Count(text[:m[2]], "\n")
			add(name, reKind.kind, line)
		}
	}
	// Class/object methods: indented identifier before '(' (skip control keywords).
	for i, line := range lines {
		if !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
			continue
		}
		m := jsMethodPattern.FindStringSubmatch(line)
		if len(m) < 2 {
			continue
		}
		name := m[1]
		switch name {
		case "if", "for", "while", "switch", "catch", "function", "class", "return":
			continue
		}
		add(name, "method", i+1)
	}
	return hits, nil
}

func extractYAMLSymbols(content []byte) ([]SymbolHit, error) {
	var root yaml.Node
	if err := yaml.Unmarshal(content, &root); err != nil {
		return nil, fmt.Errorf("treesitter yaml parse: %w", err)
	}
	doc := &root
	if root.Kind == yaml.DocumentNode && len(root.Content) > 0 {
		doc = root.Content[0]
	}
	var hits []SymbolHit
	var walk func(n *yaml.Node, depth int)
	walk = func(n *yaml.Node, depth int) {
		if n == nil || depth > 2 {
			return
		}
		if n.Kind != yaml.MappingNode {
			return
		}
		for i := 0; i+1 < len(n.Content); i += 2 {
			key := n.Content[i]
			val := n.Content[i+1]
			if key.Value != "" {
				hits = append(hits, SymbolHit{
					Name: key.Value,
					Kind: "key",
					Line: key.Line,
				})
			}
			if val != nil && val.Kind == yaml.MappingNode {
				walk(val, depth+1)
			}
		}
	}
	walk(doc, 0)
	return hits, nil
}
