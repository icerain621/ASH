import { screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { DoctorPage } from "./DoctorPage";
import { renderPage } from "@/test/renderPage";

vi.mock("@/modules/doctor/api/doctor.api", () => ({
  runDoctor: vi.fn(),
  getDoctorReport: vi.fn(),
}));

describe("DoctorPage", () => {
  it("renders doctor heading and default suite run control", () => {
    renderPage(<DoctorPage />);
    expect(screen.getByRole("heading", { name: "诊断" })).toBeInTheDocument();
    expect(screen.getByTestId("doctor-suite")).toHaveValue("TR0");
    expect(screen.getByTestId("doctor-run")).toBeInTheDocument();
    expect(screen.getByTestId("doctor-run")).toHaveAttribute("title", "运行 TR0 诊断套件");
  });
});
