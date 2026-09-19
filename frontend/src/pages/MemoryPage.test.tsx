import { fireEvent, screen, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";
import { MemoryPage } from "./MemoryPage";
import { listCandidates } from "@/modules/memory/api/memory.api";
import { renderPage } from "@/test/renderPage";

vi.mock("@/modules/memory/api/memory.api", () => ({
  listCandidates: vi.fn().mockResolvedValue({
    items: [
      { id: "mem_l2", layer: "L2", title: "L2 tip", status: "candidate" },
      { id: "mem_l0", layer: "L0", title: "L0 tip", status: "candidate" },
    ],
  }),
  queryMemory: vi.fn().mockResolvedValue({ items: [] }),
  getMemoryRecord: vi.fn(),
  getMemoryTTLQueue: vi.fn().mockResolvedValue({
    reviewDue: [],
    reviewDueCount: 0,
    expiredPendingCount: 0,
    reviewLeadDays: 7,
  }),
  createCandidate: vi.fn(),
  reviewCandidate: vi.fn(),
  sweepMemoryTTL: vi.fn(),
}));

vi.mock("@/modules/knowledge/api/knowledge.api", () => ({
  getRepoProfile: vi.fn().mockResolvedValue({
    id: "profile_test",
    repoRoot: ".",
    languages: ["go"],
    modules: ["internal"],
    testCommands: ["go test ./..."],
    summary: "languages=go",
    contextRef: "profile:profile_test",
  }),
  listWikiPages: vi.fn().mockResolvedValue({ items: [], query: "" }),
  getWikiPage: vi.fn(),
}));

vi.mock("@/modules/observability/api/observability.api", () => ({
  getRagProfile: vi.fn().mockResolvedValue({
    spaceId: "local",
    ftsAvailable: true,
    defaultRetrievalMode: "hybrid",
    hybridAvailable: true,
    lspAvailable: false,
    vectorAvailable: false,
    documentCount: 0,
    chunkCount: 0,
  }),
  rebuildRAGSymbols: vi.fn().mockResolvedValue({ paths: 0, symbols: 0, files: 0 }),
  postRagLspHover: vi.fn().mockResolvedValue({ contents: "", server: "fake" }),
  postRagLspDefinition: vi.fn().mockResolvedValue({ locations: [], server: "fake" }),
  postRagLspReferences: vi.fn().mockResolvedValue({ locations: [], source: "lsp" }),
}));

vi.mock("@tanstack/react-router", () => ({
  Link: ({
    children,
    to,
    className,
    ...rest
  }: {
    children: ReactNode;
    to?: string;
    className?: string;
    "data-testid"?: string;
  }) => (
    <a href={to ?? "#"} className={className} data-to={to} {...rest}>
      {children}
    </a>
  ),
}));

vi.mock("@/modules/interactions/api/interactions.api", () => ({
  getInteractionByRun: vi.fn(async () => ({
    runId: "run_link",
    thread: { id: "th_link", spaceId: "local", runId: "run_link", kind: "main", status: "open" },
  })),
  getInteractionThread: vi.fn(async () => ({
    threadId: "th_link",
    runId: "run_link",
    spaceId: "local",
    nodes: [],
  })),
  listInteractionMemoryLinks: vi.fn(async () => ({ threadId: "th_link", items: [] })),
  sealInteractionThread: vi.fn(),
  replayInteractionThread: vi.fn(),
}));

describe("MemoryPage", () => {
  it("renders memory console and TTL queue section", async () => {
    renderPage(<MemoryPage />);
    expect(screen.getByRole("heading", { name: "记忆" })).toBeInTheDocument();
    await waitFor(() => {
      expect(screen.getByText("TTL 复核队列")).toBeInTheDocument();
    });
  });

  it("defaults to 记忆体 tab and switches to 知识", async () => {
    renderPage(<MemoryPage />);

    expect(screen.getByTestId("memory-pillar-tabs")).toBeInTheDocument();
    expect(screen.getByTestId("memory-tab-memory")).toHaveAttribute("aria-selected", "true");
    expect(screen.getByTestId("memory-perspective")).toBeInTheDocument();
    expect(screen.queryByTestId("knowledge-panel")).not.toBeInTheDocument();

    fireEvent.click(screen.getByTestId("memory-tab-knowledge"));
    expect(screen.getByTestId("memory-tab-knowledge")).toHaveAttribute("aria-selected", "true");
    expect(screen.getByTestId("knowledge-panel")).toBeInTheDocument();
    expect(screen.queryByTestId("memory-perspective")).not.toBeInTheDocument();
    await waitFor(() => {
      expect(screen.getByTestId("knowledge-rag-hybrid")).toBeInTheDocument();
    });
  });

  it("switches perspective and shows hint when fields are missing", () => {
    renderPage(<MemoryPage />);

    expect(screen.getByTestId("memory-perspective-layer")).toHaveAttribute("aria-pressed", "true");
    expect(screen.queryByTestId("memory-perspective-hint")).not.toBeInTheDocument();
    expect(screen.getByTestId("memory-layer-filter")).toBeInTheDocument();

    fireEvent.click(screen.getByTestId("memory-perspective-skill"));
    expect(screen.getByTestId("memory-perspective-skill")).toHaveAttribute("aria-pressed", "true");
    expect(screen.getByTestId("memory-perspective-hint")).toHaveTextContent(/skill/);
    expect(screen.queryByTestId("memory-layer-filter")).not.toBeInTheDocument();
  });

  it("shows L0/L1/L2 counts on layer filters", async () => {
    renderPage(<MemoryPage />);
    await waitFor(() => {
      expect(screen.getByTestId("memory-layer-全部")).toHaveTextContent("2");
    });
    expect(screen.getByTestId("memory-layer-L0")).toHaveTextContent("1");
    expect(screen.getByTestId("memory-layer-L2")).toHaveTextContent("1");
  });

  it("groups skill perspective when skill tags exist", async () => {
    vi.mocked(listCandidates).mockResolvedValueOnce({
      items: [
        {
          id: "mem_skill",
          layer: "L1",
          title: "skill tip",
          status: "candidate",
          tags: ["skill:doctor"],
        },
      ],
      limit: 50,
      offset: 0,
      total: 1,
    });
    renderPage(<MemoryPage />);
    fireEvent.click(screen.getByTestId("memory-perspective-skill"));
    expect(await screen.findByTestId("memory-perspective-group")).toHaveTextContent("doctor");
    expect(screen.queryByTestId("memory-perspective-hint")).not.toBeInTheDocument();
  });

  it("links 去评审 to /reviews", () => {
    renderPage(<MemoryPage />);
    const link = screen.getByTestId("memory-goto-reviews");
    expect(link).toHaveTextContent("去评审");
    expect(link).toHaveAttribute("href", "/reviews");
  });

  it("opens MemoryLink tab from query params", async () => {
    const prev = window.location.search;
    window.history.replaceState({}, "", "/ui/memory?tab=links&runId=run_link");
    renderPage(<MemoryPage />);
    expect(screen.getByTestId("memory-tab-links")).toHaveAttribute("aria-selected", "true");
    expect(screen.getByTestId("memory-links-run")).toHaveValue("run_link");
    expect(await screen.findByTestId("memory-links-pane")).toBeInTheDocument();
    window.history.replaceState({}, "", `/ui/memory${prev}`);
  });

  it("switches to 关联 tab manually", () => {
    renderPage(<MemoryPage />);
    fireEvent.click(screen.getByTestId("memory-tab-links"));
    expect(screen.getByTestId("memory-tab-links")).toHaveAttribute("aria-selected", "true");
    expect(screen.getByTestId("memory-links-pane")).toBeInTheDocument();
    expect(screen.queryByTestId("memory-perspective")).not.toBeInTheDocument();
  });
});
