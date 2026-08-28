import { screen, waitFor } from "@testing-library/react";
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

describe("KnowledgePage", () => {
  it("renders knowledge heading and profile", async () => {
    renderPage(<KnowledgePage />);
    expect(screen.getByRole("heading", { name: "知识中心" })).toBeInTheDocument();
    await waitFor(() => {
      expect(screen.getByTestId("knowledge-profile")).toBeInTheDocument();
    });
    expect(screen.getByTestId("knowledge-wiki-list")).toBeInTheDocument();
  });
});
