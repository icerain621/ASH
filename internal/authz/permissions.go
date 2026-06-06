package authz

// PermissionDef describes one API/action permission in the catalog.
type PermissionDef struct {
	Key   string `json:"key"`
	Group string `json:"group"`
	Label string `json:"label"`
}

// Catalog returns the stable permission vocabulary used by RBAC.
func Catalog() []PermissionDef {
	return []PermissionDef{
		{Key: "run:create", Group: "run", Label: "创建运行"},
		{Key: "run:cancel", Group: "run", Label: "取消运行"},
		{Key: "run:approve", Group: "run", Label: "审批运行"},
		{Key: "artifact:read", Group: "artifact", Label: "读取产物"},
		{Key: "memory:create", Group: "memory", Label: "创建记忆候选"},
		{Key: "memory:read", Group: "memory", Label: "读取记忆"},
		{Key: "memory:review", Group: "memory", Label: "评审记忆"},
		{Key: "memory:query", Group: "memory", Label: "检索记忆"},
		{Key: "memory:use", Group: "memory", Label: "记录记忆命中"},
		{Key: "rag:index", Group: "rag", Label: "RAG 索引"},
		{Key: "rag:query", Group: "rag", Label: "RAG 查询"},
		{Key: "repo:read", Group: "repo", Label: "读取仓库连接"},
		{Key: "repo:write", Group: "repo", Label: "管理仓库连接"},
		{Key: "ci:read", Group: "ci", Label: "读取 CI 运行"},
		{Key: "ci:diagnose", Group: "ci", Label: "诊断 CI 失败"},
		{Key: "model:route", Group: "model", Label: "模型路由"},
		{Key: "mcp:write", Group: "mcp", Label: "注册 MCP 工具"},
		{Key: "plugin:read", Group: "plugin", Label: "读取插件"},
		{Key: "plugin:write", Group: "plugin", Label: "注册插件"},
		{Key: "audit:export", Group: "audit", Label: "导出审计"},
		{Key: "secret:read", Group: "secret", Label: "读取密钥元数据"},
		{Key: "secret:write", Group: "secret", Label: "管理密钥"},
		{Key: "member:read", Group: "org", Label: "读取成员"},
		{Key: "member:write", Group: "org", Label: "管理成员"},
		{Key: "role:read", Group: "org", Label: "读取角色"},
		{Key: "role:write", Group: "org", Label: "管理角色"},
		{Key: "space:write", Group: "org", Label: "管理空间"},
		{Key: "org:write", Group: "org", Label: "管理组织"},
		{Key: "storage:read", Group: "platform", Label: "读取存储配置"},
		{Key: "feedback:write", Group: "platform", Label: "提交反馈"},
	}
}
