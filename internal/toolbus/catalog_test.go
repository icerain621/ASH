package toolbus

import "testing"

func TestDefaultCatalogIncludesDangerRuntime(t *testing.T) {
	cat := DefaultCatalog()
	if len(cat) < 5 {
		t.Fatalf("catalog=%d want >=5 built-ins", len(cat))
	}
	byName := map[string]ToolRiskEntry{}
	for _, item := range cat {
		byName[item.Name] = item
	}
	danger, ok := byName["runtime.command"]
	if !ok || danger.Risk != RiskDanger || !danger.DefaultDeny {
		t.Fatalf("runtime.command=%+v want danger defaultDeny", danger)
	}
	if byName["test.run"].Risk != RiskSafe {
		t.Fatalf("test.run risk=%s want safe", byName["test.run"].Risk)
	}
	// Sorted by name
	for i := 1; i < len(cat); i++ {
		if cat[i-1].Name > cat[i].Name {
			t.Fatalf("catalog not sorted: %s before %s", cat[i-1].Name, cat[i].Name)
		}
	}
}
