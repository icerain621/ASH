import { screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ObservabilityPage } from "./ObservabilityPage";
import { renderPage } from "@/test/renderPage";
import { getPrometheusText, listAlertRules } from "@/modules/closure/api/closure.api";
import { getWakerQueue, getWakerStatus } from "@/modules/waker/api/waker.api";

vi.mock("@/modules/closure/api/closure.api", () => ({
  listAlerts: vi.fn().mockResolvedValue({ items: [] }),
  listAlertRules: vi.fn(),
  evaluateAlerts: vi.fn().mockResolvedValue({ evaluated: 0 }),
  putAlertRules: vi.fn(),
  getPrometheusText: vi.fn(),
  getTrace: vi.fn(),
}));

vi.mock("@/modules/platform/api/platform.api", () => ({
  getPluginHealth: vi.fn().mockResolvedValue({ items: [] }),
}));

vi.mock("@/modules/observability/api/observability.api", () => ({
  getOtelStatus: vi.fn().mockResolvedValue({ enabled: false, exporter: "none" }),
  getRagProfile: vi.fn().mockResolvedValue({
    spaceId: "local",
    defaultRetrievalMode: "fts",
    documentCount: 0,
    chunkCount: 0,
    fallbackQueryCount: 0,
    lspAvailable: true,
    vectorAvailable: false,
    vectorBackend: "qdrant",
    vectorReason: "backend unavailable",
    vectorPointCount: 0,
  }),
  postRagLspHover: vi.fn().mockResolvedValue({ contents: "hover ok", server: "fake-gopls" }),
  postRagLspDefinition: vi.fn().mockResolvedValue({ locations: [{ path: "main.go", line: 1 }], server: "fake-gopls" }),
  postRagLspReferences: vi.fn().mockResolvedValue({ locations: [], source: "lsp" }),
}));

vi.mock("@/modules/scale/api/scale.api", () => ({
  getScaleReadiness: vi.fn().mockResolvedValue({
    spaceId: "local",
    migrationReady: true,
    databaseDialect: "sqlite",
    sandboxRemoteEnabled: true,
    sandboxRemoteAvailable: true,
    sandboxRemotePreferred: true,
    sandboxRemoteDenyOnFail: false,
    sandboxRemoteBackend: "mock",
    sandboxRemoteReason: "",
  }),
}));

vi.mock("@/modules/waker/api/waker.api", () => ({
  getWakerStatus: vi.fn(),
  getWakerQueue: vi.fn(),
  listWakerDuties: vi.fn().mockResolvedValue({ duties: [] }),
  postWakerSweep: vi.fn().mockResolvedValue({ ok: true, dryRun: true, action: "report" }),
  postWakerDutyRun: vi.fn().mockResolvedValue({ ok: true, dryRun: true, action: "report" }),
  postWakerDutyEnable: vi.fn().mockResolvedValue({ id: "wd_doctor", kind: "doctor_subset", enabled: true }),
}));

const wakerStatus = {
  duties: [
    {
      id: "wd_stale",
      spaceId: "local",
      kind: "stale_run",
      enabled: true,
      intervalMs: 300000,
      nextRunAt: "2026-09-02T12:00:00Z",
    },
  ],
  recentRuns: [
    {
      id: "wdr_1",
      dutyId: "wd_stale",
      kind: "stale_run",
      status: "ok",
      matched: 2,
      flagged: 1,
      canceled: 0,
      summary: "report pass",
      startedAt: "2026-09-02T11:55:00Z",
    },
  ],
  allowCancel: false,
  interval: "5m",
  intervalMs: 300000,
  probesAvailable: true,
  alertCount: 0,
};

describe("ObservabilityPage", () => {
  beforeEach(() => {
    vi.mocked(listAlertRules).mockResolvedValue({
      items: [
        {
          id: "rule_inflight",
          name: "运行 inflight 积压",
          metric: "run_inflight_count",
          condition: "gt",
          threshold: 20,
          windowMinutes: 60,
          severity: "warn",
          enabled: true,
          description: "status=running/waiting_approval 的运行数超过阈值（Scale backlog）",
        },
      ],
    });
    vi.mocked(getPrometheusText).mockResolvedValue(
      [
        "# HELP ash_run_inflight_live",
        'ash_run_inflight_live{space_id="local"} 3',
        "ash_interaction_thread_sealed_total 2",
        "ash_interaction_replay_mismatch_total 1",
        'ash_memory_link_total{type="hit_used"} 5',
      ].join("\n") + "\n",
    );
    vi.mocked(getWakerStatus).mockResolvedValue(wakerStatus);
    vi.mocked(getWakerQueue).mockResolvedValue({
      items: [{ runId: "run_stale", spaceId: "local", status: "running", reason: "age exceeded", kind: "stale_run" }],
      count: 1,
    });
  });

  it("renders observability heading and evaluate alerts control", async () => {
    renderPage(<ObservabilityPage />);
    expect(screen.getByRole("heading", { name: "可观测与告警" })).toBeInTheDocument();
    await waitFor(() => {
      expect(screen.getByRole("button", { name: "评估告警" })).toBeInTheDocument();
      expect(screen.getByText("Space: local")).toBeInTheDocument();
    });
  });

  it("surfaces run_inflight_count in governance alert rules", async () => {
    renderPage(<ObservabilityPage />);
    await waitFor(() => {
      expect(screen.getAllByText("run_inflight_count").length).toBeGreaterThanOrEqual(1);
      expect(screen.getByText("running / waiting_approval 运行数（Scale backlog）")).toBeInTheDocument();
      expect(screen.getByText("1 条")).toBeInTheDocument();
    });
    expect(screen.getByText(/ash_run_inflight_live/)).toBeInTheDocument();
  });

  it("renders interaction derive summary from prometheus text", async () => {
    renderPage(<ObservabilityPage />);
    await waitFor(() => {
      expect(screen.getByTestId("interaction-metrics-summary")).toBeInTheDocument();
      expect(screen.getByTestId("ix-metric-sealed")).toHaveTextContent("2");
      expect(screen.getByTestId("ix-metric-mismatch")).toHaveTextContent("1");
      expect(screen.getByTestId("ix-metric-link-hit_used")).toHaveTextContent("5");
    });
  });

  it("renders Waker heading and duty kind", async () => {
    renderPage(<ObservabilityPage />);
    await waitFor(() => {
      expect(screen.getByRole("heading", { name: "Waker" })).toBeInTheDocument();
      expect(screen.getAllByText("stale_run").length).toBeGreaterThanOrEqual(1);
    });
  });

  it("surfaces review_sla duty badge and alert callout", async () => {
    vi.mocked(getWakerStatus).mockResolvedValue({
      ...wakerStatus,
      alertCount: 2,
      duties: [
        ...wakerStatus.duties,
        {
          id: "wd_review_sla",
          spaceId: "local",
          kind: "review_sla",
          enabled: true,
          intervalMs: 300000,
          nextRunAt: "2026-09-02T12:00:00Z",
        },
      ],
      recentRuns: [
        ...wakerStatus.recentRuns,
        {
          id: "wdr_sla",
          dutyId: "wd_review_sla",
          kind: "review_sla",
          status: "ok",
          matched: 3,
          flagged: 3,
          canceled: 0,
          summary: "review_sla breaches=3 hours=72 sla_breach:memory:m1",
          startedAt: "2026-09-02T11:58:00Z",
        },
      ],
    });
    renderPage(<ObservabilityPage />);
    await waitFor(() => {
      expect(screen.getAllByTestId("obs-review-sla-badge").length).toBeGreaterThanOrEqual(1);
      expect(screen.getByTestId("obs-review-sla-alert")).toHaveTextContent(
        "评审 SLA 告警：近期 duty 已标记逾期（告警 2）",
      );
    });
  });

  it("notes that cancel requires ASH_WAKER_ALLOW_CANCEL when gated off", async () => {
    renderPage(<ObservabilityPage />);
    await waitFor(() => {
      expect(screen.getByText(/ASH_WAKER_ALLOW_CANCEL=1/)).toBeInTheDocument();
    });
    expect(screen.queryByRole("button", { name: /Cancel stale/i })).not.toBeInTheDocument();
  });

  it("renders RAG LSP status and probe controls", async () => {
    renderPage(<ObservabilityPage />);
    await waitFor(() => {
      expect(screen.getByTestId("observability-rag-lsp-status")).toHaveTextContent(/可用/);
      expect(screen.getByTestId("observability-rag-lsp-probe")).toBeInTheDocument();
      expect(screen.getByTestId("observability-rag-lsp-hover")).toBeInTheDocument();
    });
  });

  it("renders remote sandbox status from scale readiness", async () => {
    renderPage(<ObservabilityPage />);
    await waitFor(() => {
      expect(screen.getByRole("heading", { name: "远程沙箱" })).toBeInTheDocument();
      expect(screen.getByTestId("observability-sandbox-remote-status")).toHaveTextContent(/后端可用/);
      expect(screen.getByTestId("observability-sandbox-remote-backend")).toHaveTextContent("mock");
      expect(screen.getByTestId("observability-sandbox-remote-policy")).toHaveTextContent(/prefer=remote/);
    });
  });

  it("renders vector backend status from rag profile", async () => {
    renderPage(<ObservabilityPage />);
    await waitFor(() => {
      expect(screen.getByTestId("observability-rag-vector-status")).toHaveTextContent(/不可用 · qdrant/);
    });
  });
});
