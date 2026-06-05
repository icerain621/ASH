import { api } from "@/services/http/client";

export type ScenarioSummary = {
  name: string;
  scenarioVersion: string;
  description?: string;
  policyProfile?: string;
  stepCount: number;
  gateCount: number;
};

export function listScenarios() {
  return api<{ items: ScenarioSummary[] }>("/scenarios");
}
