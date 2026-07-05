import { screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { MemoryPage } from "./MemoryPage";
import { renderPage } from "@/test/renderPage";

vi.mock("@/modules/memory/api/memory.api", () => ({
  listCandidates: vi.fn().mockResolvedValue({ items: [] }),
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

describe("MemoryPage", () => {
  it("renders memory console and TTL queue section", async () => {
    renderPage(<MemoryPage />);
    expect(screen.getByRole("heading", { name: "记忆" })).toBeInTheDocument();
    await waitFor(() => {
      expect(screen.getByText("TTL 复核队列")).toBeInTheDocument();
    });
  });
});
