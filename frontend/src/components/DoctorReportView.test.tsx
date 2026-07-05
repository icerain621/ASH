import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { DoctorReportView } from "./DoctorReportView";
import type { DoctorReport } from "@/modules/doctor/api/doctor.api";

const sampleReport: DoctorReport = {
  suite: "TR0",
  startedAt: 1_700_000_000_000,
  finishedAt: 1_700_000_006_000,
  summary: { pass: 1, fail: 0 },
  results: [
    {
      id: "TR0-01",
      status: "pass",
      message: "ok",
      evidence: [{ kind: "run", ref: "r1" }],
    },
  ],
};

describe("DoctorReportView", () => {
  it("shows placeholder when report is null", () => {
    render(<DoctorReportView report={null} />);
    expect(screen.getByText("尚无报告。")).toBeInTheDocument();
  });

  it("renders suite summary and localized case label", () => {
    render(<DoctorReportView report={sampleReport} />);
    expect(screen.getByText(/TR0 · 1\/1 通过/)).toBeInTheDocument();
    expect(screen.getByText("交付闭环")).toBeInTheDocument();
    expect(screen.getByText("TR0-01")).toBeInTheDocument();
  });
});
