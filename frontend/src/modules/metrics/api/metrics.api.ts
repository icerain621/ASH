import { api } from "@/services/http/client";

export type MetricCard = {
  id: string;
  label: string;
  value: number;
  unit: string;
  status: "ok" | "empty" | "unavailable" | string;
  numerator?: number;
  denominator?: number;
  description?: string;
};

export type MetricPoint = {
  periodStart: string;
  value: number;
  status: string;
};

export type MetricTrend = {
  metricId: string;
  points: MetricPoint[];
};

export type MetricBreakdown = {
  id: string;
  label: string;
  items: Array<{
    key: string;
    label: string;
    value: number;
    unit: string;
  }>;
};

export type DataQualityNote = {
  metricId: string;
  status: string;
  message: string;
};

export type MetricsOverview = {
  spaceId: string;
  projectId?: string;
  from: string;
  to: string;
  period: "day" | "week";
  summary: MetricCard[];
  trends: MetricTrend[];
  breakdowns: MetricBreakdown[];
  dataQuality: DataQualityNote[];
  generatedAt: string;
};

export type MetricsOverviewParams = {
  spaceId?: string;
  projectId?: string;
  from?: string;
  to?: string;
  period?: "day" | "week";
};

export function getMetricsOverview(params: MetricsOverviewParams = {}) {
  const qs = new URLSearchParams();
  Object.entries(params).forEach(([key, value]) => {
    if (value) qs.set(key, value);
  });
  const suffix = qs.toString() ? `?${qs.toString()}` : "";
  return api<MetricsOverview>(`/metrics/overview${suffix}`);
}

export type EvaluationSignal = {
  id: string;
  label: string;
  value: number;
  unit: string;
  numerator?: number;
  denominator?: number;
};

export type EvaluationDimension = {
  id: string;
  label: string;
  score: number;
  status: string;
  signals: EvaluationSignal[];
};

export type SpaceEvaluation = {
  spaceId: string;
  from: string;
  to: string;
  generatedAt: string;
  dimensions: EvaluationDimension[];
  health: {
    threadSealRate: EvaluationSignal;
    replayMismatchRate: EvaluationSignal;
    citationMissingTotal: number;
    memoryLinksByType?: Record<string, number>;
  };
  scenarios: Array<{
    scenario: string;
    version?: string;
    profileId: string;
    runCount: number;
    rankScore: number;
    status: string;
    dimensions: EvaluationDimension[];
    gateHints?: string[];
  }>;
  dataQuality: DataQualityNote[];
};

export type SpaceEvaluationParams = {
  from?: string;
  to?: string;
};

export function getSpaceEvaluation(spaceId: string, params: SpaceEvaluationParams = {}) {
  const qs = new URLSearchParams();
  Object.entries(params).forEach(([key, value]) => {
    if (value) qs.set(key, value);
  });
  const suffix = qs.toString() ? `?${qs.toString()}` : "";
  return api<SpaceEvaluation>(`/spaces/${encodeURIComponent(spaceId)}/evaluation${suffix}`);
}
