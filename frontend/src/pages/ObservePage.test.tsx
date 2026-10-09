import { render, screen } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, expect, it, vi } from "vitest";
import { ObservePage } from "./ObservePage";

vi.mock("@tanstack/react-router", async () => {
  const actual = await vi.importActual<typeof import("@tanstack/react-router")>("@tanstack/react-router");
  return {
    ...actual,
    useSearch: () => ({ lens: "global" }),
    Link: ({ children, ...props }: { children: React.ReactNode; to?: string; search?: unknown }) => (
      <a href={typeof props.to === "string" ? props.to : "#"} data-testid={(props as { "data-testid"?: string })["data-testid"]}>
        {children}
      </a>
    ),
  };
});

vi.mock("@/modules/observability/api/lenses.api", () => ({
  getGlobalLens: async () => ({
    generatedAt: 1,
    runningRuns: 2,
    failedClusters: [],
    templateUsage: [{ id: "tpl.react", count: 1 }],
    gateRejects: 0,
    reviewPending: 3,
    doctorChip: "BOOT",
  }),
  getAgentLens: async () => ({ runId: "r", traceId: "t", events: [] }),
  getMemoryLens: async () => ({ memoryId: "m", status: "candidate", title: "t", stages: [], events: [], redacted: true }),
}));

describe("ObservePage", () => {
  it("renders global lens chips", async () => {
    const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <QueryClientProvider client={qc}>
        <ObservePage />
      </QueryClientProvider>,
    );
    expect(screen.getByTestId("observe-page")).toBeTruthy();
    expect(await screen.findByTestId("chip-running")).toHaveTextContent("在跑 2");
    expect(screen.getByTestId("chip-review")).toHaveTextContent("待评审记忆 3");
  });
});
