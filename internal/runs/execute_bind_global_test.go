package runs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// P22: executeAgentStep must bind LegacyReactMemory on plugins.Global(), not a fresh BuiltinHost.
func TestExecuteAgentStepMemoryBindUsesGlobal(t *testing.T) {
	src, err := os.ReadFile(filepath.Join("execute.go"))
	if err != nil {
		t.Fatal(err)
	}
	body := string(src)
	if strings.Contains(body, "NewBuiltinHost().BindProduction") {
		t.Fatal("execute.go must not call NewBuiltinHost().BindProduction")
	}
	if !strings.Contains(body, "plugins.Global().BindProduction") {
		t.Fatal("execute.go must call plugins.Global().BindProduction")
	}
}
