import { api } from "@/services/http/client";

export type Space = {
  id: string;
  orgId?: string;
  name: string;
  slug?: string;
};

export type Org = {
  id: string;
  name: string;
  slug?: string;
};

export type Role = {
  id: string;
  orgId: string;
  name: string;
  permissions: string;
  createdAt?: string;
  updatedAt?: string;
};

export type Member = {
  id: string;
  orgId: string;
  spaceId: string;
  userId: string;
  roleId: string;
  status: string;
  createdAt?: string;
  updatedAt?: string;
};

export type ResourceScope = {
  id: string;
  spaceId: string;
  resourceType: string;
  resourceId: string;
  policyJson: string;
  createdAt: number;
  updatedAt: number;
};

export type ModelProvider = {
  id: string;
  provider: string;
  status: string;
  role: string;
};

export type MCPTool = {
  id: string;
  name: string;
  server: string;
  risk: string;
  status: string;
};

export type PluginRegistry = {
  id: string;
  spaceId: string;
  name: string;
  version: string;
  protocol: string;
  abi: string;
  endpoint: string;
  capabilities: string;
  compatible: boolean;
  status: string;
  lastError?: string;
  lastExportAt?: string;
  exportErrors?: number;
  dropCount?: number;
  createdAt?: string;
  updatedAt?: string;
};

export type PluginHealthSummary = {
  spaceId: string;
  pluginCount: number;
  exportErrorsTotal: number;
  dropCountTotal: number;
  staleExportCount: number;
  items: PluginRegistry[];
};

export type PluginABIProfile = {
  currentAbi: string;
  supportedAbis: string[];
  supportedProtocols: string[];
  protoPackage: string;
  goPackage: string;
  breakingPolicy: string;
  protoFiles: Array<{
    path: string;
    digest: string;
    bytes: number;
  }>;
};

export type StorageProfile = {
  database: {
    dialect: string;
    urlConfigured: boolean;
    dataDir: string;
  };
  artifactStore: {
    kind: string;
    ready: boolean;
    uri?: string;
    objectStore: boolean;
    supportsSignedUrl: boolean;
    error?: string;
  };
};

export type SecretRecord = {
  id: string;
  spaceId: string;
  name: string;
  description?: string;
  status: string;
  scope: Record<string, unknown>;
  valueDigest: string;
  redactedValue: string;
  createdBy?: string;
  updatedBy?: string;
  createdAt: string;
  updatedAt: string;
  lastUsedAt?: string;
};

export type AuditLog = {
  id: string;
  spaceId: string;
  traceId?: string;
  runId?: string;
  actorId?: string;
  eventType: string;
  payloadJSON?: string;
  createdAt: string;
};

export type AuditExport = {
  id: string;
  spaceId: string;
  status: string;
  uri: string;
  storeKey: string;
  digest: string;
  contentType: string;
  sizeBytes: number;
  requestedBy?: string;
  createdAt: string;
  completedAt?: string;
};

export type AuditExportAccess = {
  exportId: string;
  uri: string;
  signedUrl: string;
  expiresAt: number;
  digest: string;
  contentType: string;
  sizeBytes: number;
};

export type ApprovalRequest = {
  id: string;
  spaceId: string;
  runId: string;
  traceId?: string;
  stepId: string;
  gate: string;
  risk?: string;
  reason: string;
  status: string;
  requestedBy?: string;
  decidedBy?: string;
  decisionReason?: string;
  evidenceJSON?: string;
  createdAt: string;
  updatedAt?: string;
  decidedAt?: string;
};

export type AuditPolicy = {
  spaceId: string;
  retentionDays: number;
  redactPayload: boolean;
  locked: boolean;
  createdAt?: string;
  updatedAt?: string;
};

export type AuditRetentionApplyResponse = {
  spaceId: string;
  retentionDays: number;
  cutoff: string;
  matched: number;
  deleted: number;
  dryRun: boolean;
};

export type AuthMe = {
  user: {
    id: string;
    email?: string;
    displayName?: string;
  };
  space: Space;
  role: string;
  permissions: string[];
};

type RawRecord = Record<string, unknown>;

function itemsFrom<T>(res: { items?: T[]; Items?: T[] }) {
  return res.items ?? res.Items ?? [];
}

function str(raw: RawRecord, camel: string, pascal: string) {
  const value = raw[camel] ?? raw[pascal];
  return typeof value === "string" ? value : "";
}

function num(raw: RawRecord, camel: string, pascal: string) {
  const value = raw[camel] ?? raw[pascal];
  return typeof value === "number" ? value : 0;
}

function bool(raw: RawRecord, camel: string, pascal: string) {
  const value = raw[camel] ?? raw[pascal];
  return typeof value === "boolean" ? value : false;
}

function normalizeMCPTool(raw: RawRecord): MCPTool {
  return {
    id: str(raw, "id", "ID"),
    name: str(raw, "name", "Name"),
    server: str(raw, "server", "Server"),
    risk: str(raw, "risk", "Risk"),
    status: str(raw, "status", "Status"),
  };
}

function normalizeOrg(raw: RawRecord): Org {
  return {
    id: str(raw, "id", "ID"),
    name: str(raw, "name", "Name"),
    slug: str(raw, "slug", "Slug"),
  };
}

function normalizeSpace(raw: RawRecord): Space {
  return {
    id: str(raw, "id", "ID"),
    orgId: str(raw, "orgId", "OrgID"),
    name: str(raw, "name", "Name"),
    slug: str(raw, "slug", "Slug"),
  };
}

function normalizeRole(raw: RawRecord): Role {
  return {
    id: str(raw, "id", "ID"),
    orgId: str(raw, "orgId", "OrgID"),
    name: str(raw, "name", "Name"),
    permissions: str(raw, "permissions", "Permissions"),
    createdAt: str(raw, "createdAt", "CreatedAt"),
    updatedAt: str(raw, "updatedAt", "UpdatedAt"),
  };
}

function normalizeMember(raw: RawRecord): Member {
  return {
    id: str(raw, "id", "ID"),
    orgId: str(raw, "orgId", "OrgID"),
    spaceId: str(raw, "spaceId", "SpaceID"),
    userId: str(raw, "userId", "UserID"),
    roleId: str(raw, "roleId", "RoleID"),
    status: str(raw, "status", "Status"),
    createdAt: str(raw, "createdAt", "CreatedAt"),
    updatedAt: str(raw, "updatedAt", "UpdatedAt"),
  };
}

function normalizePlugin(raw: RawRecord): PluginRegistry {
  return {
    id: str(raw, "id", "ID"),
    spaceId: str(raw, "spaceId", "SpaceID"),
    name: str(raw, "name", "Name"),
    version: str(raw, "version", "Version"),
    protocol: str(raw, "protocol", "Protocol"),
    abi: str(raw, "abi", "ABI"),
    endpoint: str(raw, "endpoint", "Endpoint"),
    capabilities: str(raw, "capabilities", "Capabilities"),
    compatible: bool(raw, "compatible", "Compatible"),
    status: str(raw, "status", "Status"),
    lastError: str(raw, "lastError", "LastError"),
    lastExportAt: str(raw, "lastExportAt", "LastExportAt"),
    exportErrors: num(raw, "exportErrors", "ExportErrors"),
    dropCount: num(raw, "dropCount", "DropCount"),
    createdAt: str(raw, "createdAt", "CreatedAt"),
    updatedAt: str(raw, "updatedAt", "UpdatedAt"),
  };
}

function normalizeSecret(raw: RawRecord): SecretRecord {
  const scope = raw.scope && typeof raw.scope === "object" && !Array.isArray(raw.scope) ? raw.scope : {};
  return {
    id: str(raw, "id", "ID"),
    spaceId: str(raw, "spaceId", "SpaceID"),
    name: str(raw, "name", "Name"),
    description: str(raw, "description", "Description"),
    status: str(raw, "status", "Status"),
    scope: scope as Record<string, unknown>,
    valueDigest: str(raw, "valueDigest", "ValueDigest"),
    redactedValue: str(raw, "redactedValue", "RedactedValue") || "********",
    createdBy: str(raw, "createdBy", "CreatedBy"),
    updatedBy: str(raw, "updatedBy", "UpdatedBy"),
    createdAt: str(raw, "createdAt", "CreatedAt"),
    updatedAt: str(raw, "updatedAt", "UpdatedAt"),
    lastUsedAt: str(raw, "lastUsedAt", "LastUsedAt"),
  };
}

function normalizeAuditLog(raw: RawRecord): AuditLog {
  return {
    id: str(raw, "id", "ID"),
    spaceId: str(raw, "spaceId", "SpaceID"),
    traceId: str(raw, "traceId", "TraceID"),
    runId: str(raw, "runId", "RunID"),
    actorId: str(raw, "actorId", "ActorID"),
    eventType: str(raw, "eventType", "EventType"),
    payloadJSON: str(raw, "payloadJSON", "PayloadJSON"),
    createdAt: str(raw, "createdAt", "CreatedAt"),
  };
}

function normalizeAuditExport(raw: RawRecord): AuditExport {
  return {
    id: str(raw, "id", "ID"),
    spaceId: str(raw, "spaceId", "SpaceID"),
    status: str(raw, "status", "Status"),
    uri: str(raw, "uri", "URI"),
    storeKey: str(raw, "storeKey", "StoreKey"),
    digest: str(raw, "digest", "Digest"),
    contentType: str(raw, "contentType", "ContentType"),
    sizeBytes: num(raw, "sizeBytes", "SizeBytes"),
    requestedBy: str(raw, "requestedBy", "RequestedBy"),
    createdAt: str(raw, "createdAt", "CreatedAt"),
    completedAt: str(raw, "completedAt", "CompletedAt"),
  };
}

function normalizeApproval(raw: RawRecord): ApprovalRequest {
  return {
    id: str(raw, "id", "ID"),
    spaceId: str(raw, "spaceId", "SpaceID"),
    runId: str(raw, "runId", "RunID"),
    traceId: str(raw, "traceId", "TraceID"),
    stepId: str(raw, "stepId", "StepID"),
    gate: str(raw, "gate", "Gate"),
    risk: str(raw, "risk", "Risk"),
    reason: str(raw, "reason", "Reason"),
    status: str(raw, "status", "Status"),
    requestedBy: str(raw, "requestedBy", "RequestedBy"),
    decidedBy: str(raw, "decidedBy", "DecidedBy"),
    decisionReason: str(raw, "decisionReason", "DecisionReason"),
    evidenceJSON: str(raw, "evidenceJSON", "EvidenceJSON"),
    createdAt: str(raw, "createdAt", "CreatedAt"),
    updatedAt: str(raw, "updatedAt", "UpdatedAt"),
    decidedAt: str(raw, "decidedAt", "DecidedAt"),
  };
}

function normalizeAuditPolicy(raw: RawRecord): AuditPolicy {
  return {
    spaceId: str(raw, "spaceId", "SpaceID"),
    retentionDays: num(raw, "retentionDays", "RetentionDays"),
    redactPayload: bool(raw, "redactPayload", "RedactPayload"),
    locked: bool(raw, "locked", "Locked"),
    createdAt: str(raw, "createdAt", "CreatedAt"),
    updatedAt: str(raw, "updatedAt", "UpdatedAt"),
  };
}

export function devLogin(spaceId?: string) {
  return api<{ token: string; user: { id: string; displayName: string }; space: Space }>("/auth/dev-login", {
    method: "POST",
    body: JSON.stringify({ spaceId }),
  });
}

export function getAuthMe() {
  return api<AuthMe>("/auth/me");
}

export function listSpaces() {
  return api<{ items?: RawRecord[]; Items?: RawRecord[] }>("/spaces").then((res) => ({
    items: itemsFrom(res).map(normalizeSpace),
  }));
}

export function listOrgs() {
  return api<{ items?: RawRecord[]; Items?: RawRecord[] }>("/orgs").then((res) => ({
    items: itemsFrom(res).map(normalizeOrg),
  }));
}

export function createOrg(body: { name: string; slug?: string }) {
  return api<RawRecord>("/orgs", {
    method: "POST",
    body: JSON.stringify(body),
  }).then(normalizeOrg);
}

export type OrgTemplate = {
  id: string;
  label: string;
  description: string;
  deployment: string;
  payer: string;
  decisionMaker: string;
  approver: string;
  defaultOrgName: string;
  defaultOrgSlug: string;
  recommendedKpis: string[];
  scenarios: string[];
};

export type OrgTemplateProvisionResult = {
  templateId: string;
  org: Org;
  spaces: Space[];
  roles: Role[];
};

export function listOrgTemplates() {
  return api<{ items?: OrgTemplate[] }>("/org-templates").then((res) => ({
    items: res.items ?? [],
  }));
}

export function provisionOrgTemplate(templateId: string, body?: { name?: string; slug?: string }) {
  return api<RawRecord>(`/org-templates/${templateId}/provision`, {
    method: "POST",
    body: JSON.stringify(body ?? {}),
  }).then((raw) => {
    const org = normalizeOrg((raw.org as RawRecord) || {});
    const spacesRaw = (raw.spaces as RawRecord[] | undefined) ?? [];
    const rolesRaw = (raw.roles as RawRecord[] | undefined) ?? [];
    return {
      templateId: String(raw.templateId ?? templateId),
      org,
      spaces: spacesRaw.map(normalizeSpace),
      roles: rolesRaw.map(normalizeRole),
    } satisfies OrgTemplateProvisionResult;
  });
}

export function createSpace(body: { orgId: string; name: string; slug?: string }) {
  return api<RawRecord>("/spaces", {
    method: "POST",
    body: JSON.stringify(body),
  }).then(normalizeSpace);
}

export function listRoles(orgId: string) {
  return api<{ items?: RawRecord[]; Items?: RawRecord[] }>(`/orgs/${orgId}/roles`).then((res) => ({
    items: itemsFrom(res).map(normalizeRole),
  }));
}

export function createRole(orgId: string, body: { name: string; permissions: string[] }) {
  return api<RawRecord>(`/orgs/${orgId}/roles`, {
    method: "POST",
    body: JSON.stringify(body),
  }).then(normalizeRole);
}

export function listSpaceMembers(spaceId: string) {
  return api<{ items?: RawRecord[]; Items?: RawRecord[] }>(`/spaces/${spaceId}/members`).then((res) => ({
    items: itemsFrom(res).map(normalizeMember),
  }));
}

function normalizeResourceScope(raw: RawRecord): ResourceScope {
  return {
    id: str(raw, "id", "ID"),
    spaceId: str(raw, "spaceId", "SpaceID"),
    resourceType: str(raw, "resourceType", "ResourceType"),
    resourceId: str(raw, "resourceId", "ResourceID"),
    policyJson: str(raw, "policyJson", "PolicyJSON"),
    createdAt: num(raw, "createdAt", "CreatedAt"),
    updatedAt: num(raw, "updatedAt", "UpdatedAt"),
  };
}

export function listSpaceResourceScopes(spaceId: string) {
  return api<{ items?: RawRecord[]; Items?: RawRecord[] }>(`/spaces/${spaceId}/resource-scopes`).then(
    (res) => ({
      items: itemsFrom(res).map(normalizeResourceScope),
    }),
  );
}

export function updateSpaceResourceScope(spaceId: string, scopeId: string, policyJson: string) {
  return api<RawRecord>(`/spaces/${spaceId}/resource-scopes/${scopeId}`, {
    method: "PUT",
    body: JSON.stringify({ policyJson }),
  }).then(normalizeResourceScope);
}

export function createSpaceMember(
  spaceId: string,
  body: { userId?: string; email?: string; displayName?: string; password?: string; roleId: string; status?: string },
) {
  return api<RawRecord>(`/spaces/${spaceId}/members`, {
    method: "POST",
    body: JSON.stringify(body),
  }).then(normalizeMember);
}

export function listModelProviders() {
  return api<{ items: ModelProvider[] }>("/model-router/providers");
}

export type ToolRiskEntry = {
  name: string;
  risk: string;
  defaultDeny: boolean;
  label: string;
};

export function listToolRiskCatalog() {
  return api<{ items?: ToolRiskEntry[]; docRef?: string }>("/tools/risk-catalog").then((res) => ({
    items: res.items ?? [],
    docRef: res.docRef ?? "",
  }));
}

export function listMCPTools() {
  return api<{ items?: RawRecord[]; Items?: RawRecord[] }>("/mcp/tools").then((res) => ({
    items: itemsFrom(res).map(normalizeMCPTool),
  }));
}

export function listPlugins() {
  return api<{ items?: RawRecord[]; Items?: RawRecord[] }>("/plugins").then((res) => ({
    items: itemsFrom(res).map(normalizePlugin),
  }));
}

export function getPluginHealth() {
  return api<RawRecord>("/plugins/health").then((raw) => {
    const items = itemsFrom(raw as { items?: RawRecord[]; Items?: RawRecord[] });
    return {
      spaceId: str(raw, "spaceId", "SpaceID") ?? "",
      pluginCount: num(raw, "pluginCount", "PluginCount") ?? 0,
      exportErrorsTotal: num(raw, "exportErrorsTotal", "ExportErrorsTotal") ?? 0,
      dropCountTotal: num(raw, "dropCountTotal", "DropCountTotal") ?? 0,
      staleExportCount: num(raw, "staleExportCount", "StaleExportCount") ?? 0,
      items: items.map(normalizePlugin),
    } satisfies PluginHealthSummary;
  });
}

export function reportPluginExport(pluginId: string, body: { ok?: boolean; dropped?: number } = {}) {
  return api<RawRecord>(`/plugins/${pluginId}/export-report`, {
    method: "POST",
    body: JSON.stringify(body),
  }).then(normalizePlugin);
}

export function verifyPlugin(pluginId: string) {
  return api<RawRecord>(`/plugins/${pluginId}/verify`, {
    method: "POST",
    body: "{}",
  }).then(normalizePlugin);
}

export function getPluginABIProfile() {
  return api<PluginABIProfile>("/plugins/abi");
}

export function getStorageProfile() {
  return api<StorageProfile>("/storage/profile");
}

export function listSecrets() {
  return api<{ items?: RawRecord[]; Items?: RawRecord[] }>("/secrets").then((res) => ({
    items: itemsFrom(res).map(normalizeSecret),
  }));
}

export function createSecret(body: {
  name: string;
  value: string;
  description?: string;
  scope?: Record<string, unknown>;
}) {
  return api<RawRecord>("/secrets", {
    method: "POST",
    body: JSON.stringify(body),
  }).then(normalizeSecret);
}

export function rotateSecret(secretId: string, body: { value: string; description?: string }) {
  return api<RawRecord>(`/secrets/${secretId}/rotate`, {
    method: "POST",
    body: JSON.stringify(body),
  }).then(normalizeSecret);
}

export function deleteSecret(secretId: string) {
  return api<void>(`/secrets/${secretId}`, {
    method: "DELETE",
  });
}

export function listAuditLogs(params: { q?: string; eventType?: string; runId?: string; limit?: number } = {}) {
  const search = new URLSearchParams();
  if (params.q) search.set("q", params.q);
  if (params.eventType) search.set("eventType", params.eventType);
  if (params.runId) search.set("runId", params.runId);
  search.set("limit", String(params.limit ?? 50));
  return api<{ items?: RawRecord[]; Items?: RawRecord[] }>(`/audit/logs?${search.toString()}`).then((res) => ({
    items: itemsFrom(res).map(normalizeAuditLog),
  }));
}

export function listAuditExports() {
  return api<{ items?: RawRecord[]; Items?: RawRecord[] }>("/audit/exports").then((res) => ({
    items: itemsFrom(res).map(normalizeAuditExport),
  }));
}

export function createAuditExport() {
  return api<RawRecord>("/audit/export", {
    method: "POST",
    body: "{}",
  }).then(normalizeAuditExport);
}

export function getAuditExportAccess(exportId: string) {
  return api<AuditExportAccess>(`/audit/exports/${exportId}/access?ttlSeconds=900`);
}

export function listApprovals(params: { status?: string; limit?: number } = {}) {
  const search = new URLSearchParams();
  search.set("status", params.status ?? "pending");
  search.set("limit", String(params.limit ?? 50));
  return api<{ items?: RawRecord[]; Items?: RawRecord[] }>(`/approvals?${search.toString()}`).then((res) => ({
    items: itemsFrom(res).map(normalizeApproval),
  }));
}

export function approveApproval(approvalId: string, body: { actorId?: string; reason?: string }) {
  return api<{ runId: string; ok: boolean }>(`/approvals/${approvalId}/approve`, {
    method: "POST",
    body: JSON.stringify(body),
  });
}

export function rejectApproval(approvalId: string, body: { actorId?: string; reason?: string }) {
  return api<{ runId: string; status: string }>(`/approvals/${approvalId}/reject`, {
    method: "POST",
    body: JSON.stringify(body),
  });
}

export function getAuditPolicy() {
  return api<RawRecord>("/audit/policy").then(normalizeAuditPolicy);
}

export function updateAuditPolicy(body: { retentionDays: number; redactPayload: boolean }) {
  return api<RawRecord>("/audit/policy", {
    method: "PUT",
    body: JSON.stringify(body),
  }).then(normalizeAuditPolicy);
}

export function applyAuditRetention(body: { dryRun?: boolean }) {
  return api<AuditRetentionApplyResponse>("/audit/retention/apply", {
    method: "POST",
    body: JSON.stringify(body),
  });
}

export type PermissionDef = { key: string; group: string; label: string };
export type BuiltinRole = { name: string; label: string; permissions: string[] };
export type ToolRule = { allow?: string[]; deny?: string[]; denyMode?: string };
export type ScenarioMatrixRow = {
  scenarioKey: string;
  scenario: string;
  version: string;
  toolMatrix: Record<string, ToolRule>;
};
export type PermissionMatrix = {
  spaceId: string;
  catalog: PermissionDef[];
  builtinRoles: BuiltinRole[];
  orgRoles?: { id: string; name: string; permissions: string[] }[];
  scenarioTools: ScenarioMatrixRow[];
  currentRole?: string;
  currentActor?: string;
};

export function getPermissionMatrix(spaceId?: string) {
  const path = spaceId
    ? `/spaces/${encodeURIComponent(spaceId)}/permissions/matrix`
    : "/permissions/matrix";
  return api<PermissionMatrix>(path);
}

export function createFeedback(body: {
  targetType: string;
  targetId: string;
  rating?: number;
  comment?: string;
}) {
  return api<unknown>("/feedback", {
    method: "POST",
    body: JSON.stringify(body),
  });
}
