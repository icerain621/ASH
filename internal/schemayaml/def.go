package schemayaml

import (
	"bytes"
	"fmt"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

var defCache sync.Map // cacheKey -> *compiled

// ValidateJSONDef validates a JSON document against a named $defs entry in schemaJSON.
// Returns nil when defName is not declared (unknown event types are allowed).
func ValidateJSONDef(schemaJSON []byte, schemaID, defName string, document any) []Issue {
	cacheKey := schemaID + "#" + defName
	c := cachedDefSchema(schemaJSON, schemaID, defName, cacheKey)
	if c == nil {
		return nil
	}
	if c.err != nil {
		return []Issue{{Path: "$", Code: "SCHEMA_LOAD_ERROR", Message: c.err.Error()}}
	}
	if err := c.schema.Validate(document); err != nil {
		return issuesFromError(err)
	}
	return nil
}

func cachedDefSchema(schemaJSON []byte, schemaID, defName, cacheKey string) *compiled {
	if v, ok := defCache.Load(cacheKey); ok {
		return v.(*compiled)
	}
	root, err := jsonschema.UnmarshalJSON(bytes.NewReader(schemaJSON))
	if err != nil {
		c := &compiled{err: err}
		defCache.Store(cacheKey, c)
		return c
	}
	rootMap, ok := root.(map[string]any)
	if !ok {
		c := &compiled{err: fmt.Errorf("schema root is not an object")}
		defCache.Store(cacheKey, c)
		return c
	}
	defs, ok := rootMap["$defs"].(map[string]any)
	if !ok {
		c := &compiled{err: fmt.Errorf("schema missing $defs")}
		defCache.Store(cacheKey, c)
		return c
	}
	sub, ok := defs[defName]
	if !ok {
		return nil
	}
	subMap, ok := sub.(map[string]any)
	if !ok {
		c := &compiled{err: fmt.Errorf("def %q is not an object", defName)}
		defCache.Store(cacheKey, c)
		return c
	}
	standalone := map[string]any{
		"$schema": rootMap["$schema"],
		"$id":     schemaID + "/defs/" + defName,
	}
	for k, v := range subMap {
		standalone[k] = v
	}
	c := &compiled{}
	defCache.Store(cacheKey, c)
	c.once.Do(func() {
		compiler := jsonschema.NewCompiler()
		id := standalone["$id"].(string)
		if err := compiler.AddResource(id, standalone); err != nil {
			c.err = err
			return
		}
		c.schema, c.err = compiler.Compile(id)
	})
	return c
}

// ValidateJSON checks a JSON document against a full schema document.
func ValidateJSON(schemaJSON []byte, schemaID string, document any) []Issue {
	c := cachedSchema(schemaJSON, schemaID)
	if c.err != nil {
		return []Issue{{Path: "$", Code: "SCHEMA_LOAD_ERROR", Message: c.err.Error()}}
	}
	if err := c.schema.Validate(document); err != nil {
		return issuesFromError(err)
	}
	return nil
}

// KnownDefs returns $defs keys declared in schemaJSON.
func KnownDefs(schemaJSON []byte) ([]string, error) {
	root, err := jsonschema.UnmarshalJSON(bytes.NewReader(schemaJSON))
	if err != nil {
		return nil, err
	}
	rootMap, ok := root.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("schema root is not an object")
	}
	defs, ok := rootMap["$defs"].(map[string]any)
	if !ok {
		return nil, nil
	}
	out := make([]string, 0, len(defs))
	for name := range defs {
		out = append(out, name)
	}
	return out, nil
}
