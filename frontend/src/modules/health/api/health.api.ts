export type ReadyzResponse = {
  status: string;
  region?: string;
  dialect?: string;
  error?: string;
  schemaMode?: string;
  sqlMigrationVersion?: number;
  sqlMigrationExpected?: number;
  postgresRLSEnabled?: boolean;
  postgresRLSPolicyCount?: number;
  postgresRLSPolicyExpected?: number;
  rlsCatalogSummary?: string;
  readinessWarnings?: string[];
  liveGateHints?: string[];
  otelEnabled?: boolean;
  alertsEvalInterval?: string;
  memoryTTLSweepInterval?: string;
  metricsEventReplayEnabled?: boolean;
  retentionEventsDays?: number;
  retentionAuditDays?: number;
  retentionArtifactsDays?: number;
  retentionArtifactsMaxRuns?: number;
};

export async function getReadyz(): Promise<ReadyzResponse> {
  const res = await fetch("/readyz");
  const text = await res.text();
  let data: ReadyzResponse;
  try {
    data = JSON.parse(text) as ReadyzResponse;
  } catch {
    throw new Error(text || res.statusText || "readyz parse failed");
  }
  if (!res.ok) {
    throw new Error(data.error || res.statusText || "readyz not ready");
  }
  return data;
}
