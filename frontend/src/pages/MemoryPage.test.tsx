import { fireEvent, screen, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";
import { MemoryPage } from "./MemoryPage";
import {
  createCandidate,
  getMemoryTTLQueue,
  listCandidates,
  queryMemory,
  sweepMemoryTTL,
} from "@/modules/memory/api/memory.api";
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
  sweepMemoryTTL: vi.fn().mockResolvedValue({ ok: true, deprecated: 0, reviewDue: 0 }),
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

  it("links candidate 厚审 to reviews memory queue", async () => {
    renderPage(<MemoryPage />);
    const link = await screen.findByTestId("memory-thick-review-mem_l2");
    expect(link).toHaveAttribute("href", "/ui/reviews?queue=memory&memoryId=mem_l2");
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
    });
    renderPage(<MemoryPage />);
    fireEvent.click(screen.getByTestId("memory-perspective-skill"));
    expect(await screen.findByTestId("memory-perspective-group")).toHaveTextContent("doctor");
    expect(screen.queryByTestId("memory-perspective-hint")).not.toBeInTheDocument();
  });

  it("restores the candidate form after a successful create", async () => {
    vi.mocked(createCandidate).mockResolvedValue({ candidateId: "mem_spot" });
    const confirmSpy = vi.spyOn(window, "confirm").mockReturnValue(true);
    renderPage(<MemoryPage />);
    const title = screen.getByDisplayValue("M1 治理规则");
    fireEvent.change(title, { target: { value: "点测标题" } });
    fireEvent.click(screen.getByRole("button", { name: "提交候选" }));
    await waitFor(() => {
      expect(screen.getByDisplayValue("M1 治理规则")).toBeInTheDocument();
    });
    expect(screen.queryByDisplayValue("点测标题")).not.toBeInTheDocument();
    confirmSpy.mockRestore();
  });

  it("disables candidate submit until title and body are filled", () => {
    renderPage(<MemoryPage />);
    const submit = screen.getByTestId("memory-candidate-submit");
    expect(submit).toBeEnabled();
    expect(submit).toHaveAttribute("title", "提交记忆候选（需确认）");
    fireEvent.change(screen.getByTestId("memory-candidate-title"), { target: { value: "" } });
    expect(submit).toBeDisabled();
    expect(submit).toHaveAttribute("title", "需要填写标题");
    fireEvent.change(screen.getByTestId("memory-candidate-title"), { target: { value: "有标题" } });
    fireEvent.change(screen.getByTestId("memory-candidate-body"), { target: { value: "" } });
    expect(submit).toBeDisabled();
    expect(submit).toHaveAttribute("title", "需要填写内容");
  });

  it("requires confirm before submitting a memory candidate", async () => {
    vi.mocked(createCandidate).mockClear();
    const confirmSpy = vi.spyOn(window, "confirm").mockReturnValue(false);
    renderPage(<MemoryPage />);
    fireEvent.click(screen.getByTestId("memory-candidate-submit"));
    expect(confirmSpy).toHaveBeenCalled();
    expect(createCandidate).not.toHaveBeenCalled();
    confirmSpy.mockRestore();
  });

  it("shows the empty search row when query items is null", async () => {
    vi.mocked(queryMemory).mockResolvedValueOnce({ items: null as unknown as [] });
    renderPage(<MemoryPage />);
    const queryBtn = screen.getByTestId("memory-query-submit");
    expect(queryBtn).toBeEnabled();
    expect(queryBtn).toHaveAttribute("title", "检索记忆");
    fireEvent.click(queryBtn);
    expect(await screen.findByText("尚无检索结果。")).toBeInTheDocument();
  });

  it("disables memory query until keyword is filled", () => {
    renderPage(<MemoryPage />);
    const input = screen.getByTestId("memory-query-input");
    fireEvent.change(input, { target: { value: "   " } });
    const queryBtn = screen.getByTestId("memory-query-submit");
    expect(queryBtn).toBeDisabled();
    expect(queryBtn).toHaveAttribute("title", "需要填写关键词");
  });

  it("links 去评审 to memory review queue", () => {
    renderPage(<MemoryPage />);
    const link = screen.getByTestId("memory-goto-reviews");
    expect(link).toHaveTextContent("去评审");
    expect(link.getAttribute("href") || "").toMatch(/queue=memory/);
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

  it("offers TTL dry-run and requires confirm before live sweep", async () => {
    vi.mocked(getMemoryTTLQueue).mockResolvedValue({
      reviewDue: [],
      reviewDueCount: 0,
      expiredPendingCount: 2,
      reviewLeadDays: 7,
    });
    vi.mocked(sweepMemoryTTL).mockResolvedValue({ ok: true, deprecated: 2, reviewDue: 0 });
    renderPage(<MemoryPage />);
    const dry = await screen.findByTestId("memory-ttl-sweep-dry");
    const live = screen.getByTestId("memory-ttl-sweep");
    expect(live).toHaveAttribute("title", "弃用过期记忆（不可恢复，需确认）");

    fireEvent.click(dry);
    await waitFor(() => {
      expect(sweepMemoryTTL).toHaveBeenCalledWith({ dryRun: true });
    });

    const confirmSpy = vi.spyOn(window, "confirm").mockReturnValue(false);
    fireEvent.click(live);
    expect(confirmSpy).toHaveBeenCalled();
    expect(sweepMemoryTTL).not.toHaveBeenCalledWith({ dryRun: false });

    confirmSpy.mockReturnValue(true);
    fireEvent.click(live);
    await waitFor(() => {
      expect(sweepMemoryTTL).toHaveBeenCalledWith({ dryRun: false });
    });
    confirmSpy.mockRestore();
    vi.mocked(getMemoryTTLQueue).mockResolvedValue({
      reviewDue: [],
      reviewDueCount: 0,
      expiredPendingCount: 0,
      reviewLeadDays: 7,
    });
  });

  it("requires confirm before approving or rejecting a memory candidate", async () => {
    const { reviewCandidate } = await import("@/modules/memory/api/memory.api");
    vi.mocked(reviewCandidate).mockResolvedValue({ ok: true } as never);
    const confirmSpy = vi.spyOn(window, "confirm").mockReturnValue(false);
    renderPage(<MemoryPage />);
    const approve = await screen.findByTestId("memory-candidate-approve-mem_l2");
    expect(approve).toHaveAttribute("title", "通过候选（需确认）");
    fireEvent.click(approve);
    expect(confirmSpy).toHaveBeenCalled();
    expect(reviewCandidate).not.toHaveBeenCalled();

    const reject = screen.getByTestId("memory-candidate-reject-mem_l2");
    expect(reject).toHaveAttribute("title", "拒绝候选（需确认）");
    fireEvent.click(reject);
    expect(reviewCandidate).not.toHaveBeenCalled();

    confirmSpy.mockReturnValue(true);
    fireEvent.click(approve);
    await waitFor(() => {
      expect(reviewCandidate).toHaveBeenCalledWith("mem_l2", expect.objectContaining({ decision: "approve" }));
    });
    confirmSpy.mockRestore();
  });
});
