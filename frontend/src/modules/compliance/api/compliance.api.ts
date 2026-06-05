import { api } from "@/services/http/client";
import type { DoctorReport } from "@/modules/doctor/api/doctor.api";

export type LeakFinding = {
  source: string;
  ref: string;
  pattern: string;
  snippet: string;
};

export type SecretScanResult = {
  spaceId: string;
  scanned: number;
  leakCount: number;
  redactEnabled: boolean;
  findings: LeakFinding[];
};

export function scanSecrets(limit = 200) {
  return api<SecretScanResult>(`/compliance/secret-scan?limit=${limit}`);
}

export function exportComplianceBundle(body: { suite?: string; reportId?: string }) {
  return api<{ exportId: string; reportId?: string; suite: string; report?: DoctorReport }>(
    "/compliance/export",
    {
      method: "POST",
      body: JSON.stringify(body),
    },
  );
}
