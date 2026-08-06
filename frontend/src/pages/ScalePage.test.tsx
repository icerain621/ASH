import { screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { ScalePage } from "./ScalePage";
import { renderPage } from "@/test/renderPage";

vi.mock("@/modules/scale/api/scale.api", () => ({
  getScaleReadiness: vi.fn().mockResolvedValue({
    spaceId: "local",
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
});
