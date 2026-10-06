import { screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { renderPage } from "@/test/renderPage";
import { NotFoundPage } from "../pages/NotFoundPage";
import { router } from "./router";

vi.mock("@tanstack/react-router", async () => {
  const actual = await vi.importActual<typeof import("@tanstack/react-router")>("@tanstack/react-router");
  return {
    ...actual,
    Link: ({
      children,
      to,
      className,
      ...rest
    }: {
      children: React.ReactNode;
      to?: string;
      className?: string;
      "data-testid"?: string;
    }) => (
      <a href={to ? `/ui${to}` : "#"} className={className} {...rest}>
        {children}
      </a>
    ),
  };
});

describe("console route aliases", () => {
  it("registers legacy shorthand paths that previously rendered a blank shell", () => {
    const paths = Object.keys(router.routesByPath ?? {}).map((p) => p.replace(/\/+$/, ""));
    for (const alias of [
      "/improve",
      "/improves",
      "/skills",
      "/approvals",
      "/secrets",
      "/tools",
      "/compare",
      "/registry",
      "/rag",
      "/chat",
      "/plan",
      "/otel",
      "/mobile/reviews",
    ]) {
      expect(paths, `missing alias ${alias}`).toContain(alias);
    }
    for (const dest of [
      "/automation",
      "/reviews",
      "/knowledge",
      "/agent",
      "/runs",
      "/m/reviews",
      "/observability",
    ]) {
      expect(paths, `missing destination ${dest}`).toContain(dest);
    }
  });

  it("registers a splat catch-all for unknown paths", () => {
    const paths = Object.keys(router.routesByPath ?? {});
    expect(paths.some((p) => p.includes("$"))).toBe(true);
  });
});

describe("NotFoundPage", () => {
  it("renders heading and home link", () => {
    renderPage(<NotFoundPage />);
    expect(screen.getByTestId("console-not-found")).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "页面不存在" })).toBeInTheDocument();
    expect(screen.getByTestId("console-not-found-home")).toHaveAttribute("href", "/ui/quest");
  });
});
