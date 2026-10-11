package plugins

import "testing"

func TestSetGlobalForTest(t *testing.T) {
	ResetGlobalForTest()
	t.Cleanup(ResetGlobalForTest)

	empty := NewEmptyHost()
	SetGlobalForTest(empty)
	if Global() != empty {
		t.Fatal("SetGlobalForTest did not install host")
	}
	if err := Global().BindProduction([]string{MemRetrieve}); err == nil {
		t.Fatal("empty host must reject builtin ids as unknown")
	}
	ResetGlobalForTest()
	if err := Global().BindProduction([]string{MemRetrieve, MemInject}); err != nil {
		t.Fatalf("builtin Global: %v", err)
	}
}
