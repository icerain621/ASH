import { api } from "@/services/http/client";

export function runDoctor(suite = "TR0") {
  return api<{ reportId: string }>("/doctor/run", {
    method: "POST",
    body: JSON.stringify({ suite, format: "json" }),
  });
}

export function getDoctorReport(reportId: string) {
  return api<unknown>(`/doctor/reports/${reportId}`);
}
