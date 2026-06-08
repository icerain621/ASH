import { api } from "@/services/http/client";

export type OtelStatus = {
  enabled: boolean;
  serviceName: string;
  endpoint?: string;
  insecure?: boolean;
};

export function getOtelStatus() {
  return api<OtelStatus>("/observability/otel/status");
}

export type RagProfile = {
  spaceId: string;
  ftsAvailable: boolean;
  ftsEngine?: string;
  defaultRetrievalMode: string;
  databaseDialect?: string;
  documentCount: number;
  chunkCount: number;
  fallbackQueryCount: number;
};

export function getRagProfile() {
  return api<RagProfile>("/rag/profile");
}
