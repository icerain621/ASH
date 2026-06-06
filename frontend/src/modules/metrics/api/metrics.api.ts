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
