import { fireEvent, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { renderPage } from "@/test/renderPage";
import { LoginPage } from "./LoginPage";

vi.mock("@tanstack/react-router", () => ({
  Link: ({
    children,
    to,
    className,
  }: {
    children: React.ReactNode;
    to?: string;
    className?: string;
  }) => (
    <a href={to ? `/ui${to}` : "#"} className={className}>
      {children}
    </a>
  ),
  useNavigate: () => vi.fn(),
}));

vi.mock("@/modules/health/api/health.api", () => ({
  getReadyz: vi.fn().mockResolvedValue({ status: "ready", consoleAuthRequired: false }),
}));

vi.mock("@/modules/platform/api/platform.api", () => ({
  passwordLogin: vi.fn().mockResolvedValue({
    token: "tok",
    refreshToken: "ref",
    space: { id: "local", name: "Local" },
  }),
}));

describe("LoginPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("disables submit until email and non-blank password are set", async () => {
    renderPage(<LoginPage />);
    const submit = await screen.findByTestId("password-login-submit");
    expect(submit).toBeDisabled();
    expect(submit).toHaveAttribute("title", "需要填写邮箱");

    fireEvent.change(screen.getByTestId("login-email"), { target: { value: "a@b.c" } });
    expect(submit).toBeDisabled();
    expect(submit).toHaveAttribute("title", "需要填写密码");

    fireEvent.change(screen.getByTestId("login-password"), { target: { value: "   " } });
    expect(submit).toBeDisabled();
    expect(submit).toHaveAttribute("title", "需要填写密码");

    fireEvent.change(screen.getByTestId("login-password"), { target: { value: "secret" } });
    expect(submit).not.toBeDisabled();
    expect(submit).toHaveAttribute("title", "密码登录");
  });

  it("does not call passwordLogin when password is blank", async () => {
    const platform = await import("@/modules/platform/api/platform.api");
    renderPage(<LoginPage />);
    fireEvent.change(await screen.findByTestId("login-email"), { target: { value: "a@b.c" } });
    fireEvent.change(screen.getByTestId("login-password"), { target: { value: "   " } });
    fireEvent.submit(screen.getByTestId("password-login-form"));
    await waitFor(() => {
      expect(platform.passwordLogin).not.toHaveBeenCalled();
    });
  });
});
