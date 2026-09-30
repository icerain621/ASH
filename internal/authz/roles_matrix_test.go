package authz

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/ash-repwiki/ash/internal/store"
)

func TestCatalogUniqueKeys(t *testing.T) {
	seen := map[string]struct{}{}
	for _, p := range Catalog() {
		if p.Key == "" || p.Group == "" {
			t.Fatalf("invalid permission: %+v", p)
		}
		if _, ok := seen[p.Key]; ok {
			t.Fatalf("duplicate key %q", p.Key)
		}
		seen[p.Key] = struct{}{}
	}
	if len(seen) < 20 {
		t.Fatalf("catalog too small: %d", len(seen))
	}
}

func TestRoleAllowsWildcards(t *testing.T) {
	if !RoleAllows("admin", "secret:write") {
		t.Fatal("admin * should allow secret:write")
	}
	if !RoleAllows("maintainer", "run:approve") {
		t.Fatal("maintainer run:* should allow run:approve")
	}
	if !RoleAllows("maintainer", "memory:manage") {
		t.Fatal("maintainer memory:* should allow memory:manage")
	}
	if RoleAllows("developer", "memory:review") {
		t.Fatal("developer must not memory:review")
	}
	if RoleAllows("unknown-role", "artifact:read") {
		t.Fatal("unknown role must deny")
	}
	if RoleAllows("viewer", "") {
		t.Fatal("empty permission must deny")
	}
}

func TestPermissionMatchPatterns(t *testing.T) {
	cases := []struct {
		grant, want string
		ok          bool
	}{
		{"*", "anything", true},
		{"run:create", "run:create", true},
		{"run:*", "run:cancel", true},
		{"run:*", "memory:read", false},
		{"", "run:create", false},
		{"run:create", "", false},
		{"run:create", "run:cancel", false},
	}
	for _, tc := range cases {
		if got := permissionMatch(tc.grant, tc.want); got != tc.ok {
			t.Fatalf("permissionMatch(%q,%q)=%v want %v", tc.grant, tc.want, got, tc.ok)
		}
	}
}

func TestToolPatternMatch(t *testing.T) {
	if !toolPatternMatch("*", "git.status") {
		t.Fatal("want * match")
	}
	if !toolPatternMatch("git.*", "git.status") {
		t.Fatal("want git.* prefix")
	}
	if toolPatternMatch("git.*", "test.run") {
		t.Fatal("git.* must not match test.run")
	}
	if toolPatternMatch("", "git.status") {
		t.Fatal("empty pattern denies")
	}
	if !toolPatternMatch("apply_patch", "apply_patch") {
		t.Fatal("exact match")
	}
}

func TestParsePermissionList(t *testing.T) {
	got := parsePermissionList(`["a","b"]`)
	if len(got) != 2 || got[0] != "a" {
		t.Fatalf("json list: %+v", got)
	}
	got = parsePermissionList("single")
	if len(got) != 1 || got[0] != "single" {
		t.Fatalf("raw: %+v", got)
	}
	if parsePermissionList("") != nil {
		t.Fatal("empty want nil")
	}
}

func TestBuildMatrixNilDB(t *testing.T) {
	resp, err := BuildMatrix(nil, "local", "", "viewer", "u1")
	if err != nil {
		t.Fatal(err)
	}
	if resp.SpaceID != "local" || resp.CurrentRole != "viewer" || resp.CurrentActor != "u1" {
		t.Fatalf("resp=%+v", resp)
	}
	if len(resp.Catalog) == 0 || len(resp.BuiltinRoles) == 0 {
		t.Fatal("want catalog and builtin roles")
	}
	if len(resp.OrgRoles) != 0 {
		t.Fatalf("org roles=%d want 0", len(resp.OrgRoles))
	}
}

func TestBuildMatrixWithOrgRoles(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	now := time.Now().UTC()
	perms, _ := json.Marshal([]string{"run:create", "artifact:read"})
	if err := db.Create(&store.Role{
		ID: "role_custom", OrgID: "org_1", Name: "custom", Permissions: string(perms), CreatedAt: now, UpdatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}
	resp, err := BuildMatrix(db, "local", "org_1", "maintainer", "actor")
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.OrgRoles) != 1 || resp.OrgRoles[0].Name != "custom" {
		t.Fatalf("orgRoles=%+v", resp.OrgRoles)
	}
	if len(resp.OrgRoles[0].Permissions) != 2 {
		t.Fatalf("perms=%+v", resp.OrgRoles[0].Permissions)
	}
}

func TestParseScenarioToolPolicyInvalid(t *testing.T) {
	if _, err := ParseScenarioToolPolicy("{"); err == nil {
		t.Fatal("want parse error")
	}
	pol, err := ParseScenarioToolPolicy("")
	if err != nil {
		t.Fatal(err)
	}
	if len(pol.ToolMatrix) != 0 {
		t.Fatalf("empty policy matrix=%+v", pol.ToolMatrix)
	}
}

func TestFirstNonEmpty(t *testing.T) {
	if got := firstNonEmpty("", "  ", "x", "y"); got != "x" {
		t.Fatalf("got %q", got)
	}
	if got := firstNonEmpty("", " "); got != "" {
		t.Fatalf("got %q", got)
	}
}
