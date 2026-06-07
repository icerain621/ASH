package schemayaml

import (
	"bytes"
	"fmt"
	"strings"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"gopkg.in/yaml.v3"
)

// Issue describes a schema validation error with JSON-pointer-like path.
type Issue struct {
	Path    string `json:"path"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type compiled struct {
	once   sync.Once
	schema *jsonschema.Schema
	err    error
}

var cache sync.Map // schemaID -> *compiled

// Validate checks raw YAML against an embedded JSON Schema document.
func Validate(schemaJSON []byte, schemaID string, raw []byte) []Issue {
	c := cachedSchema(schemaJSON, schemaID)
	if c.err != nil {
		return []Issue{{Path: "$", Code: "SCHEMA_LOAD_ERROR", Message: c.err.Error()}}
	}
	var doc any
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return []Issue{{Path: "$", Code: "YAML_PARSE_ERROR", Message: err.Error()}}
	}
	jsonDoc, err := nodeToJSON(doc)
	if err != nil {
		return []Issue{{Path: "$", Code: "YAML_TO_JSON_ERROR", Message: err.Error()}}
	}
	if err := c.schema.Validate(jsonDoc); err != nil {
		return issuesFromError(err)
	}
	return nil
}

func cachedSchema(schemaJSON []byte, schemaID string) *compiled {
	v, _ := cache.LoadOrStore(schemaID, &compiled{})
	c := v.(*compiled)
	c.once.Do(func() {
		doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(schemaJSON))
		if err != nil {
			c.err = err
			return
		}
		compiler := jsonschema.NewCompiler()
		if err := compiler.AddResource(schemaID, doc); err != nil {
			c.err = err
			return
		}
		c.schema, c.err = compiler.Compile(schemaID)
	})
	return c
}

func issuesFromError(err error) []Issue {
	ve, ok := err.(*jsonschema.ValidationError)
	if !ok {
		return []Issue{{Path: "$", Code: "SCHEMA_INVALID", Message: err.Error()}}
	}
	out := make([]Issue, 0, len(ve.Causes)+1)
	for _, cause := range ve.Causes {
		out = append(out, Issue{
			Path:    pathFromLocation(cause.InstanceLocation),
			Code:    "SCHEMA_INVALID",
			Message: strings.TrimSpace(cause.Error()),
		})
	}
	if len(out) == 0 {
		out = append(out, Issue{
			Path:    pathFromLocation(ve.InstanceLocation),
			Code:    "SCHEMA_INVALID",
			Message: strings.TrimSpace(ve.Error()),
		})
	}
	return out
}

func pathFromLocation(loc []string) string {
	if len(loc) == 0 {
		return "$"
	}
	return "$." + strings.Join(loc, ".")
}

func nodeToJSON(v any) (any, error) {
	switch t := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, val := range t {
			converted, err := nodeToJSON(val)
			if err != nil {
				return nil, err
			}
			out[k] = converted
		}
		return out, nil
	case map[any]any:
		out := make(map[string]any, len(t))
		for k, val := range t {
			converted, err := nodeToJSON(val)
			if err != nil {
				return nil, err
			}
			out[fmt.Sprint(k)] = converted
		}
		return out, nil
	case []any:
		out := make([]any, len(t))
		for i, val := range t {
			converted, err := nodeToJSON(val)
			if err != nil {
				return nil, err
			}
			out[i] = converted
		}
		return out, nil
	default:
		return v, nil
	}
}
