package authz

import "testing"

func TestEvaluateScenarioToolReviewerDeniesPatch(t *testing.T) {
	policy := DefaultScenarioPolicyJSON("feature_delivery", "1.0.0")
	ok, _ := EvaluateScenarioTool(policy, "reviewer", "git.status")
	if !ok {
		t.Fatal("reviewer should read git.status")
	}
	ok, reason := EvaluateScenarioTool(policy, "reviewer", "apply_patch")
	if ok {
		t.Fatal("reviewer must not apply_patch")
	}
	if reason == "" {
		t.Fatal("want deny reason")
	}
}

func TestEvaluateScenarioToolSecurityPatchOperator(t *testing.T) {
	policy := DefaultScenarioPolicyJSON("security_patch", "1.1.0")
	ok, _ := EvaluateScenarioTool(policy, "operator", "test.run")
	if !ok {
		t.Fatal("operator should run tests")
	}
	ok, _ = EvaluateScenarioTool(policy, "operator", "apply_patch")
	if ok {
		t.Fatal("operator must not apply_patch in security_patch")
	}
}

func TestRoleAllowsViewerReadOnly(t *testing.T) {
	if !RoleAllows("viewer", "artifact:read") {
		t.Fatal("viewer should read artifacts")
	}
	if RoleAllows("viewer", "run:create") {
		t.Fatal("viewer must not create runs")
	}
}
