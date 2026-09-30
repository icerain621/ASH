package rules

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEngineNilDocument(t *testing.T) {
	e := NewEngine(nil)
	if e.Document() != nil {
		t.Fatal("want nil document")
	}
	if e.RequiredInputs() != nil {
		t.Fatal("want nil inputs")
	}
	if e.RequiredArtifactTypes() != nil {
		t.Fatal("want nil artifacts")
	}
	if e.GatesBeforeStep("x") != nil || e.GatesBeforeFinish() != nil {
		t.Fatal("want nil gates")
	}
	if denied, _ := e.EvaluateHooks("any", nil); denied {
		t.Fatal("nil eng must not deny")
	}
	if e.StepOrder() != nil {
		t.Fatal("want nil step order")
	}
}

func TestEngineGatesAndOrder(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "scenarios", "feature_delivery.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	res := ParseAndValidate(raw)
	if !res.OK {
		t.Fatalf("validate: %+v", res.Issues)
	}
	e := NewEngine(res.Doc)
	if e.Document() == nil {
		t.Fatal("want document")
	}
	inputs := e.RequiredInputs()
	if len(inputs) == 0 {
		t.Fatal("want required inputs")
	}
	arts := e.RequiredArtifactTypes()
	if len(arts) == 0 {
		t.Fatal("want required artifacts")
	}
	order := e.StepOrder()
	if len(order) < 2 {
		t.Fatalf("step order=%v", order)
	}
	_ = e.GatesBeforeStep(order[0])
	_ = e.GatesBeforeFinish()
	if denied, _ := e.EvaluateHooks("nonexistent.event", map[string]any{"x": 1}); denied {
		t.Fatal("unknown event must not deny")
	}
}

func TestLoaderGetAndList(t *testing.T) {
	dir := filepath.Join("..", "..", "scenarios")
	loader := NewLoader(dir)
	if err := loader.LoadDir(); err != nil {
		t.Fatal(err)
	}
	list := loader.List()
	if len(list) == 0 {
		t.Fatal("want scenarios")
	}
	doc, err := loader.Get(list[0].Name, list[0].ScenarioVersion)
	if err != nil || doc == nil {
		t.Fatalf("get: %v doc=%v", err, doc)
	}
	if _, err := loader.Get("missing", "0.0.0"); err == nil {
		t.Fatal("want missing error")
	}
	raw, err := loader.RawYAML(list[0].Name, list[0].ScenarioVersion)
	if err != nil || len(raw) == 0 {
		t.Fatalf("raw: %v", err)
	}
	vres := loader.ValidateYAML(raw)
	if !vres.OK {
		t.Fatalf("validate yaml: %+v", vres.Issues)
	}
}

func TestMatchAll(t *testing.T) {
	if !matchAll(map[string]any{"a": 1}, map[string]any{"a": 1, "b": 2}) {
		t.Fatal("want match")
	}
	if matchAll(map[string]any{"a": 1}, map[string]any{"a": 2}) {
		t.Fatal("want mismatch")
	}
	if matchAll(map[string]any{"a": 1}, map[string]any{}) {
		t.Fatal("missing key")
	}
}
