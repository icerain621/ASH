import { render, screen, within } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

const routerState = vi.hoisted(() => ({ pathname: "/quest" }));

vi.mock("@tanstack/react-router", async () => {
  const React = await import("react");
  return {
    Link: ({
      to,
      children,
      className,
      onClick,
      ...rest
    }: {
      to: string;
      children?: React.ReactNode;
      className?: string;
      onClick?: () => void;
      [key: string]: unknown;
    }) =>
      React.createElement(
        "a",
        { href: to, className, onClick, ...rest },
        children,
      ),
    Outlet: () => React.createElement("div", { "data-testid": "outlet" }),
    useRouterState: ({ select }: { select: (s: { location: { pathname: string } }) => unknown }) =>
      select({ location: { pathname: routerState.pathname } }),
  };
});

vi.mock("@/services/http/client", () => ({
  getCurrentSpaceId: () => "local",
}));

import { AppLayout } from "./AppLayout";

describe("AppLayout three-pillar shell", () => {
  beforeEach(() => {
    routerState.pathname = "/quest";
  });

  it("renders three pillar buttons and settings dropdown", () => {
    render(<AppLayout />);

    expect(screen.getByTestId("work-mode-agent")).toHaveTextContent("Agent");
    expect(screen.getByTestId("work-mode-memory")).toHaveTextContent("记忆");
    expect(screen.getByTestId("work-mode-review")).toHaveTextContent("评审管控");

    const settings = screen.getByTestId("nav-settings");
    expect(within(settings).getByLabelText("设置")).toBeInTheDocument();
    expect(within(settings).queryByText("设置")).not.toBeInTheDocument();
    expect(within(settings).getByTestId("nav-settings-workspace-hdr")).toHaveTextContent(
      "账号与工作区",
    );
    expect(within(settings).getByTestId("nav-settings-ops-hdr")).toHaveTextContent("运维与合规");
    expect(within(settings).getByTestId("nav-settings-space")).toHaveTextContent("空间设置");
    expect(within(settings).getByTestId("nav-settings-quest")).toHaveTextContent("任务板");
    expect(within(settings).getByText("运行")).toBeInTheDocument();
    expect(within(settings).getByText("自动化")).toBeInTheDocument();
    expect(within(settings).getByText("反馈")).toBeInTheDocument();
    expect(within(settings).getByText("CI")).toBeInTheDocument();
    expect(within(settings).getByText("发布")).toBeInTheDocument();
    expect(within(settings).getByText("合规")).toBeInTheDocument();
    expect(within(settings).getByText("规模化")).toBeInTheDocument();
    expect(within(settings).getByText("诊断")).toBeInTheDocument();
    expect(within(settings).getByText("指标")).toBeInTheDocument();
    expect(within(settings).getByText("可观测")).toBeInTheDocument();

    const account = screen.getByTestId("nav-account");
    expect(within(account).getByTestId("nav-account-space")).toHaveTextContent("local");
    expect(within(account).getByTestId("nav-login")).toHaveTextContent("登录");
    expect(within(account).getByTestId("nav-space")).toHaveTextContent("Space 设置");
  });

  it("does not render flat top-level tabs for secondary pages", () => {
    const { container } = render(<AppLayout />);
    expect(container.querySelector("nav.tabs")).toBeNull();

    const header = container.querySelector("header.header");
    expect(header).not.toBeNull();
    const settings = screen.getByTestId("nav-settings");
    expect(within(settings).getByText("规模化")).toBeInTheDocument();
    expect(header!.querySelectorAll(":scope > nav.tabs .tab")).toHaveLength(0);
  });

  it("keeps mobile shell without desktop header", () => {
    routerState.pathname = "/m/reviews";
    render(<AppLayout />);
    expect(screen.getByTestId("mobile-shell")).toBeInTheDocument();
    expect(screen.queryByTestId("work-mode-switch")).not.toBeInTheDocument();
  });
});
