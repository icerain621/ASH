package openapicheck

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// SchemaPropertyNames returns property keys for an OpenAPI 3 components.schemas entry.
func SchemaPropertyNames(path, schemaName string) ([]string, error) {
	root, err := readYAML(path)
	if err != nil {
		return nil, err
	}
	comps, _ := root["components"].(map[string]any)
	if comps == nil {
		return nil, fmt.Errorf("%s: missing components", path)
	}
	schemas, _ := comps["schemas"].(map[string]any)
	if schemas == nil {
		return nil, fmt.Errorf("%s: missing components.schemas", path)
	}
	raw, ok := schemas[schemaName]
	if !ok {
		return nil, fmt.Errorf("%s: schema %q not found", path, schemaName)
	}
	return propertyNamesFromSchema(raw)
}

// SwaggerDefinitionPropertyNames returns property keys for a swag definitions entry.
func SwaggerDefinitionPropertyNames(path, definitionName string) ([]string, error) {
	root, err := readYAML(path)
	if err != nil {
		return nil, err
	}
	defs, _ := root["definitions"].(map[string]any)
	if defs == nil {
		return nil, fmt.Errorf("%s: missing definitions", path)
	}
	raw, ok := defs[definitionName]
	if !ok {
		return nil, fmt.Errorf("%s: definition %q not found", path, definitionName)
	}
	return propertyNamesFromSchema(raw)
}

// DiffPropertyNames reports keys present in only one of the two sorted name lists.
func DiffPropertyNames(contract, swagger []string) (missingInContract, missingInSwagger []string) {
	cSet := toSet(contract)
	sSet := toSet(swagger)
	for name := range sSet {
		if _, ok := cSet[name]; !ok {
			missingInContract = append(missingInContract, name)
		}
	}
	for name := range cSet {
		if _, ok := sSet[name]; !ok {
			missingInSwagger = append(missingInSwagger, name)
		}
	}
	sort.Strings(missingInContract)
	sort.Strings(missingInSwagger)
	return missingInContract, missingInSwagger
}

func readYAML(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var root map[string]any
	if err := yaml.Unmarshal(data, &root); err != nil {
		return nil, err
	}
	return root, nil
}

func propertyNamesFromSchema(raw any) ([]string, error) {
	obj, ok := raw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("schema is not an object")
	}
	props, _ := obj["properties"].(map[string]any)
	if props == nil {
		return nil, fmt.Errorf("schema has no properties")
	}
	out := make([]string, 0, len(props))
	for name := range props {
		out = append(out, name)
	}
	sort.Strings(out)
	return out, nil
}

func toSet(names []string) map[string]struct{} {
	out := make(map[string]struct{}, len(names))
	for _, name := range names {
		out[strings.TrimSpace(name)] = struct{}{}
	}
	return out
}
