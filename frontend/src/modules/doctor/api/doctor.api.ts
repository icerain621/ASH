import { api } from "@/services/http/client";

export type DoctorEvidence = {
  kind: string;
  ref: string;
  digest?: string;
};

export type DoctorCaseResult = {
  id: string;
  status: string;
  runId?: string;
  message?: string;
  evidence?: DoctorEvidence[];
};

export type DoctorSuite = "TR0" | "TR1" | "TR2" | "TR3" | "M2" | "M3" | "ALL";

export type DoctorReport = {
  suite: string;
  startedAt: number;
  finishedAt: number;
  results: DoctorCaseResult[];
  summary: { pass: number; fail: number };
};

export function runDoctor(suite: DoctorSuite = "TR0") {
  return api<{ reportId: string }>("/doctor/run", {
    method: "POST",
    body: JSON.stringify({ suite, format: "json" }),
  });
}

export function getDoctorReport(reportId: string) {
  return api<DoctorReport>(`/doctor/reports/${reportId}`);
}
