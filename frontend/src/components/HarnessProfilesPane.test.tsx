import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { HarnessProfilesPane } from "./HarnessProfilesPane";
import { listHarnessProfiles, loadActiveHarnessProfile } from "@/modules/reviews/api/reviews.api";

vi.mock("@/modules/reviews/api/reviews.api", () => ({
  listHarnessProfiles: vi.fn(async () => ({
    items: [],
    sandboxBackends: ["local", "landlock", "docker", "remote-mock", "remote-e2b"],
  })),
  loadActiveHarnessProfile: vi.fn(async () => ({})),
  createHarnessProfile: vi.fn(),
  submitHarnessReview: vi.fn(),
  promoteHarnessProfile: vi.fn(),
  rollbackHarnessProfile: vi.fn(),
}));

describe("HarnessProfilesPane", () => {
  it("shows the sandbox backend catalog from the list envelope", async () => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <QueryClientProvider client={client}>
        <HarnessProfilesPane />
      </QueryClientProvider>,
    );
    await waitFor(() => {
      expect(screen.getByTestId("harness-sandbox-backends")).toHaveTextContent(
        "沙箱目录 · local · landlock · docker · remote-mock · remote-e2b",
      );
    });
    expect(listHarnessProfiles).toHaveBeenCalled();
    expect(loadActiveHarnessProfile).toHaveBeenCalled();
  });
});
