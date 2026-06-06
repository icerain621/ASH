import { api } from "@/services/http/client";

export type ScaleReadiness = {
  spaceId: string;
  memorySchemaVersion: number;
  memoryApprovedCount: number;
  ragDocumentCount: number;
  ragChunkCount: number;
  modelUsageRows: number;
  modelCostMicrosTotal: number;
  qualityMetricRows: number;
  auditLogRows: number;
  databaseDialect?: string;
  postgresConfigured?: boolean;
  migrationReady?: boolean;
  sqlitePath?: string;
  migrationTableCount?: number;
  dualWriteEnabled?: boolean;
  dualWriteRuntime?: boolean;
  dualWriteSource?: string;
  lastMigrationSyncAtMs?: number;
  postgresRLSEnabled?: boolean;
  postgresRLSForce?: boolean;
  postgresRLSPolicyCount?: number;
  postgresAppUrlConfigured?: boolean;
};

export function getScaleReadiness() {
  return api<ScaleReadiness>("/scale/readiness");
}
