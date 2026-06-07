package openapicheck

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// OperationMeta is summary/tag metadata from swag output.
type OperationMeta struct {
	Summary string
	Tags    []string
}

// LoadSwaggerOperations reads path/method metadata from swag swagger.yaml.
func LoadSwaggerOperations(path string) (map[string]map[string]OperationMeta, error) {
	data, err := swaggerReadFile(path)
	if err != nil {
		return nil, err
	}
	var root map[string]any
	if err := yaml.Unmarshal(data, &root); err != nil {
		return nil, err
	}
	rawPaths, _ := root["paths"].(map[string]any)
	out := map[string]map[string]OperationMeta{}
	for p, raw := range rawPaths {
		ops, _ := raw.(map[string]any)
		out[p] = map[string]OperationMeta{}
		for k, rawOp := range ops {
			kl := strings.ToLower(k)
			if !httpMethods[kl] {
				continue
			}
			op, _ := rawOp.(map[string]any)
			meta := OperationMeta{}
			if s, _ := op["summary"].(string); s != "" {
				meta.Summary = s
			}
			if tags, _ := op["tags"].([]any); len(tags) > 0 {
				for _, t := range tags {
					if ts, _ := t.(string); ts != "" {
						meta.Tags = append(meta.Tags, titleTag(ts))
					}
				}
			}
			out[p][kl] = meta
		}
	}
	return out, nil
}

func titleTag(tag string) string {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return "Platform"
	}
	parts := strings.Split(tag, "-")
	for i, p := range parts {
		if p == "" {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	return strings.Join(parts, "")
}

var pathParamRe = regexp.MustCompile(`\{([^}]+)\}`)

// EmitMissingContractYAML renders minimal OpenAPI path entries for swagger ops
// not yet present in the hand-written contract.
func EmitMissingContractYAML(contract PathMethods, swagger map[string]map[string]OperationMeta) string {
	type item struct {
		path   string
		method string
		meta   OperationMeta
	}
	var items []item
	for path, methods := range swagger {
		if !strings.HasPrefix(path, "/api/v1/") {
			continue
		}
		for method, meta := range methods {
			if contract[path] != nil {
				if _, ok := contract[path][method]; ok {
					continue
				}
			}
			items = append(items, item{path: path, method: method, meta: meta})
		}
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].path == items[j].path {
			return items[i].method < items[j].method
		}
		return items[i].path < items[j].path
	})

	var b strings.Builder
	curPath := ""
	for _, it := range items {
		if it.path != curPath {
			if curPath != "" {
				b.WriteString("\n")
			}
			fmt.Fprintf(&b, "  %s:\n", it.path)
			curPath = it.path
		}
		tag := "Platform"
		if len(it.meta.Tags) > 0 {
			tag = it.meta.Tags[0]
		}
		summary := it.meta.Summary
		if summary == "" {
			summary = fmt.Sprintf("%s %s", strings.ToUpper(it.method), it.path)
		}
		fmt.Fprintf(&b, "    %s:\n", it.method)
		fmt.Fprintf(&b, "      tags: [%s]\n", tag)
		fmt.Fprintf(&b, "      summary: %s\n", yamlQuote(summary))
		if params := pathParamRe.FindAllStringSubmatch(it.path, -1); len(params) > 0 {
			fmt.Fprintf(&b, "      parameters:\n")
			for _, name := range params {
				fmt.Fprintf(&b, "        - in: path\n")
				fmt.Fprintf(&b, "          name: %s\n", name[1])
				fmt.Fprintf(&b, "          required: true\n")
				fmt.Fprintf(&b, "          schema: { type: string }\n")
			}
		}
		status := "200"
		if it.method == "delete" {
			status = "204"
		}
		fmt.Fprintf(&b, "      responses:\n")
		fmt.Fprintf(&b, "        \"%s\":\n", status)
		if it.method == "delete" {
			fmt.Fprintf(&b, "          description: no content\n")
		} else {
			fmt.Fprintf(&b, "          description: ok\n")
			fmt.Fprintf(&b, "          content:\n")
			fmt.Fprintf(&b, "            application/json:\n")
			fmt.Fprintf(&b, "              schema: { $ref: \"#/components/schemas/ApiResponse\" }\n")
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

func yamlQuote(s string) string {
	if strings.ContainsAny(s, ":\"'\n") {
		return fmt.Sprintf("%q", s)
	}
	return s
}

// swaggerReadFile is a seam for tests.
var swaggerReadFile = os.ReadFile
