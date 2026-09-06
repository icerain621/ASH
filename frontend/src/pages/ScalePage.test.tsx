import { screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { ScalePage } from "./ScalePage";
import { renderPage } from "@/test/renderPage";

vi.mock("@/modules/scale/api/scale.api", () => ({
  getScaleReadiness: vi.fn().mockResolvedValue({
    spaceId: "local",
    region: "default",
    memorySchemaVersion: 2,
    migrationReady: true,
    databaseDialect: "sqlite",
    runRunningCount: 2,
    runWaitingApprovalCount: 1,
    runInflightCount: 3,
  }),
}));

vi.mock("@/modules/platform/api/platform.api", () => ({
  getPluginHealth: vi.fn().mockResolvedValue({ items: [] }),
}));

vi.mock("@/modules/health/api/health.api", () => ({
  getReadyz: vi.fn().mockResolvedValue({
    status: "ready",
    region: "default",
    dialect: "sqlite",
    sqlMigrationExpected: 20,
  }),
}));

vi.mock("@/modules/doctor/api/doctor.api", () => ({
  runDoctor: vi.fn(),
  getDoctorReport: vi.fn(),
}));

vi.mock("@/modules/memory/api/memory.api", () => ({
  runMemoryMigration: vi.fn(),
  sweepMemoryTTL: vi.fn(),
}));

vi.mock("@/modules/waker/api/waker.api", () => ({
  getWakerStatus: vi.fn().mockResolvedValue({
    duties: [
      {
        id: "wd_stale",
        spaceId: "local",
        kind: "stale_run",
        enabled: true,
        intervalMs: 300000,
        nextRunAt: "2026-09-02T12:00:00Z",
      },
      {
        id: "wd_ttl",
        spaceId: "local",
        kind: "memory_ttl",
        enabled: false,
        intervalMs: 600000,
        nextRunAt: "2026-09-02T12:00:00Z",
      },
    ],
    recentRuns: [],
    allowCancel: false,
    interval: "5m",
    intervalMs: 300000,
  }),
  getWakerQueue: vi.fn().mockResolvedValue({ items: [], count: 3 }),
}));

describe("ScalePage", () => {
  it("renders scale readiness heading and M3 checklist", async () => {
    renderPage(<ScalePage />);
    expect(screen.getByRole("heading", { name: "规模化就绪" })).toBeInTheDocument();
    await waitFor(() => {
      expect(screen.getByText("M3-01")).toBeInTheDocument();
      expect(screen.getByText("TR3-10")).toBeInTheDocument();
    });
  });

  it("shows run inflight backlog counts from readiness", async () => {
    renderPage(<ScalePage />);
    await waitFor(() => {
      expect(screen.getByText("运行积压（inflight）")).toBeInTheDocument();
      expect(screen.getByText(/3（running 2 · waiting 1）/)).toBeInTheDocument();
    });
  });

  it("shows compact waker counts from status and queue", async () => {
    renderPage(<ScalePage />);
    await waitFor(() => {
      expect(screen.getByText("duties enabled: 1 · queue: 3 · ticker 5m")).toBeInTheDocument();
    });
  });
});
