import { screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";
import { DoctorPage } from "./DoctorPage";
import { renderPage } from "@/test/renderPage";

vi.mock("@/modules/doctor/api/doctor.api", () => ({
  runDoctor: vi.fn(),
  getDoctorReport: vi.fn(),
}));

vi.mock("@tanstack/react-router", () => ({
  Link: ({ children, ...props }: { children: ReactNode; to?: string; className?: string }) => (
    <a href={props.to ?? "#"} className={props.className} data-testid={(props as { "data-testid"?: string })["data-testid"]}>
      {children}
    </a>
  ),
}));

describe("DoctorPage", () => {
  it("renders doctor heading and default suite run control", () => {
    renderPage(<DoctorPage />);
    expect(screen.getByRole("heading", { name: "诊断" })).toBeInTheDocument();
    expect(screen.getByTestId("doctor-suite")).toHaveValue("BOOT");
    expect(screen.getByTestId("doctor-run")).toBeInTheDocument();
    expect(screen.getByTestId("doctor-run")).toHaveAttribute("title", "运行 BOOT 诊断套件");
    expect(screen.getByTestId("doctor-quality-redirect")).toBeInTheDocument();
  });
});
