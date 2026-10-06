import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { KnowledgePage } from "./KnowledgePage";
import { renderPage } from "@/test/renderPage";

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
  listWikiPages: vi.fn().mockResolvedValue({
    items: [
      {
        id: "wiki_profile_overview",
        title: "Repo Profile Overview",
        body: "overview",
        source: "synthetic",
        contextRef: "wiki:wiki_profile_overview",
      },
    ],
    query: "architecture",
  }),
  getWikiPage: vi.fn().mockResolvedValue({
    id: "wiki_profile_overview",
    title: "Repo Profile Overview",
    body: "overview body",
    source: "synthetic",
    contextRef: "wiki:wiki_profile_overview",
  }),
}));

vi.mock("@/modules/observability/api/observability.api", () => ({
  getRagProfile: vi.fn().mockResolvedValue({
    spaceId: "local",
    ftsAvailable: true,
    defaultRetrievalMode: "hybrid",
    hybridAvailable: true,
    lspAvailable: true,
    vectorAvailable: true,
    vectorBackend: "mock",
    vectorPointCount: 2,
    documentCount: 2,
    chunkCount: 4,
    pathEntryCount: 2,
    symbolCount: 3,
    fallbackQueryCount: 0,
  }),
  rebuildRAGSymbols: vi.fn().mockResolvedValue({ paths: 2, symbols: 3, files: 2 }),
  postRagLspHover: vi.fn().mockResolvedValue({ contents: "ok", server: "fake" }),
  postRagLspDefinition: vi.fn().mockResolvedValue({ locations: [], server: "fake" }),
  postRagLspReferences: vi.fn().mockResolvedValue({ locations: [], source: "lsp" }),
}));

describe("KnowledgePage", () => {
  it("renders knowledge heading and profile", async () => {
    renderPage(<KnowledgePage />);
    expect(screen.getByRole("heading", { name: "知识中心" })).toBeInTheDocument();
    await waitFor(() => {
      expect(screen.getByTestId("knowledge-profile")).toBeInTheDocument();
    });
    expect(screen.getByTestId("knowledge-wiki-list")).toBeInTheDocument();
    expect(screen.getByTestId("knowledge-rag-hybrid")).toBeInTheDocument();
    expect(screen.getByTestId("knowledge-rag-vector")).toBeInTheDocument();
    expect(screen.getByText(/LSP 可用/)).toBeInTheDocument();
    expect(screen.getByText(/向量：可用/)).toBeInTheDocument();
    expect(screen.getByTestId("knowledge-rag-rebuild")).toBeInTheDocument();
    expect(screen.getByTestId("knowledge-rag-lsp-probe")).toBeInTheDocument();
  });

  it("disables RAG rebuild when repoRoot is empty", async () => {
    renderPage(<KnowledgePage />);
    await waitFor(() => {
      expect(screen.getByTestId("knowledge-rag-rebuild")).toBeInTheDocument();
    });
    const rebuild = screen.getByTestId("knowledge-rag-rebuild");
    expect(rebuild).not.toBeDisabled();
    fireEvent.change(screen.getByTestId("knowledge-repo-root"), { target: { value: "" } });
    expect(rebuild).toBeDisabled();
    expect(rebuild).toHaveAttribute("title", "需要填写 repoRoot");
  });

  it("requires confirm before rebuilding RAG index", async () => {
    const { rebuildRAGSymbols } = await import("@/modules/observability/api/observability.api");
    const confirmSpy = vi.spyOn(window, "confirm").mockReturnValue(false);
    renderPage(<KnowledgePage />);
    const rebuild = await screen.findByTestId("knowledge-rag-rebuild");
    expect(rebuild).toHaveAttribute("title", "重建符号/路径索引（可能耗时，需确认）");
    fireEvent.click(rebuild);
    expect(confirmSpy).toHaveBeenCalled();
    expect(rebuildRAGSymbols).not.toHaveBeenCalled();
    confirmSpy.mockRestore();
  });
});
