import { fireEvent, screen, waitFor } from "@testing-library/react";
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
  getRagProfile: vi.fn().mockResolvedValue({ retrievalMode: "fts", documentCount: 0 }),
}));

vi.mock("@/modules/scale/api/scale.api", () => ({
  getScaleReadiness: vi.fn().mockResolvedValue({
    spaceId: "local",
    migrationReady: true,
    databaseDialect: "sqlite",
  }),
}));

vi.mock("@/modules/waker/api/waker.api", () => ({
  getWakerStatus: vi.fn(),
  getWakerQueue: vi.fn(),
  listWakerDuties: vi.fn().mockResolvedValue({ duties: [] }),
  postWakerSweep: vi.fn().mockResolvedValue({ ok: true, dryRun: true, action: "report" }),
  postWakerDutyRun: vi.fn().mockResolvedValue({ ok: true, dryRun: true, action: "report" }),
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
      '# HELP ash_run_inflight_live\nash_run_inflight_live{space_id="local"} 3\n',
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

  it("renders Waker heading and duty kind", async () => {
    renderPage(<ObservabilityPage />);
    await waitFor(() => {
      expect(screen.getByRole("heading", { name: "Waker" })).toBeInTheDocument();
      expect(screen.getAllByText("stale_run").length).toBeGreaterThanOrEqual(1);
    });
  });

  it("notes that cancel requires ASH_WAKER_ALLOW_CANCEL when gated off", async () => {
    renderPage(<ObservabilityPage />);
    await waitFor(() => {
      expect(screen.getByText(/ASH_WAKER_ALLOW_CANCEL=1/)).toBeInTheDocument();
    });
    expect(screen.queryByRole("button", { name: /Cancel stale/i })).not.toBeInTheDocument();
  });

  it("enables Cancel stale only after exact CANCEL_STALE_RUNS confirm", async () => {
    vi.mocked(getWakerStatus).mockResolvedValue({ ...wakerStatus, allowCancel: true });
    renderPage(<ObservabilityPage />);
    await waitFor(() => {
      expect(screen.getByRole("button", { name: /Cancel stale/i })).toBeDisabled();
    });
    fireEvent.change(screen.getByPlaceholderText("CANCEL_STALE_RUNS"), { target: { value: "CANCEL_STALE_RUNS" } });
    expect(screen.getByRole("button", { name: /Cancel stale/i })).toBeEnabled();
  });
});
