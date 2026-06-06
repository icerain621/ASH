package authz

// BuiltinRole is a built-in RBAC role template.
type BuiltinRole struct {
	Name        string   `json:"name"`
	Label       string   `json:"label"`
	Permissions []string `json:"permissions"`
}

// BuiltinRoles returns default role → permission mappings (M2 matrix baseline).
func BuiltinRoles() []BuiltinRole {
	return []BuiltinRole{
		{Name: "admin", Label: "管理员", Permissions: []string{"*"}},
		{
			Name:  "maintainer",
			Label: "维护者",
			Permissions: []string{
				"run:*", "memory:*", "rag:*", "model:route", "plugin:*",
				"artifact:read", "storage:read", "feedback:write", "mcp:write",
				"repo:*", "ci:*",
			},
		},
		{
			Name:  "developer",
			Label: "开发者",
			Permissions: []string{
				"run:create", "run:cancel", "artifact:read", "rag:query",
				"repo:read", "ci:read", "ci:diagnose", "feedback:write",
			},
		},
		{
			Name:        "operator",
			Label:       "操作员",
			Permissions: []string{"run:create", "run:cancel", "artifact:read", "rag:query", "repo:read", "ci:read", "ci:diagnose"},
		},
		{
			Name:        "reviewer",
			Label:       "评审员",
			Permissions: []string{"memory:review", "run:approve", "artifact:read"},
		},
		{Name: "auditor", Label: "审计员", Permissions: []string{"audit:export", "artifact:read", "repo:read", "ci:read", "ci:diagnose"}},
		{
			Name:  "viewer",
			Label: "只读",
			Permissions: []string{
				"artifact:read", "memory:read", "memory:query", "rag:query",
				"plugin:read", "storage:read", "role:read", "member:read",
				"repo:read", "ci:read",
			},
		},
	}
}

// RoleAllows reports whether a built-in role name grants a permission key.
func RoleAllows(role, permission string) bool {
	for _, item := range BuiltinRoles() {
		if item.Name != role {
			continue
		}
		for _, grant := range item.Permissions {
			if permissionMatch(grant, permission) {
				return true
			}
		}
		return false
	}
	return false
}
