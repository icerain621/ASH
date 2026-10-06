import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { RagLspProbePanel } from "./RagLspProbePanel";
import { postRagLspHover } from "@/modules/observability/api/observability.api";

vi.mock("@/modules/observability/api/observability.api", () => ({
  postRagLspHover: vi.fn().mockResolvedValue({ contents: "ok", server: "fake" }),
  postRagLspDefinition: vi.fn().mockResolvedValue({ locations: [], server: "fake" }),
  postRagLspReferences: vi.fn().mockResolvedValue({ locations: [], source: "lsp" }),
}));

vi.mock("@/services/http/client", () => ({
  getCurrentSpaceId: () => "local",
}));

function renderProbe(props: Partial<Parameters<typeof RagLspProbePanel>[0]> = {}) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
  return render(
    <QueryClientProvider client={qc}>
      <RagLspProbePanel testIdPrefix="probe" {...props} />
    </QueryClientProvider>,
  );
}

describe("RagLspProbePanel", () => {
  beforeEach(() => {
    vi.mocked(postRagLspHover).mockClear();
  });

  it("defaults path to cmd/worker/main.go instead of bare main.go", () => {
    renderProbe();
    expect(screen.getByTestId("probe-path")).toHaveValue("cmd/worker/main.go");
  });

  it("disables actions when path is cleared", () => {
    renderProbe();
    fireEvent.change(screen.getByTestId("probe-path"), { target: { value: "" } });
    expect(screen.getByTestId("probe-hover")).toBeDisabled();
    expect(screen.getByTestId("probe-hover")).toHaveAttribute("title", "需要填写 path");
  });

  it("disables actions when repoRoot is cleared", () => {
    renderProbe();
    fireEvent.change(screen.getByTestId("probe-repo-root"), { target: { value: "" } });
    expect(screen.getByTestId("probe-hover")).toBeDisabled();
    expect(screen.getByTestId("probe-hover")).toHaveAttribute("title", "需要填写 repoRoot");
  });

  it("exposes action titles when path is ready", () => {
    renderProbe();
    expect(screen.getByTestId("probe-hover")).toHaveAttribute("title", "LSP Hover（当前 path/line/char）");
    expect(screen.getByTestId("probe-definition")).toHaveAttribute("title", "LSP Definition（跳转定义）");
    expect(screen.getByTestId("probe-references")).toHaveAttribute("title", "LSP References（查找引用）");
  });

  it("disables actions when line is below 1 or not an integer", () => {
    renderProbe();
    const hover = screen.getByTestId("probe-hover");
    fireEvent.change(screen.getByTestId("probe-line"), { target: { value: "0" } });
    expect(hover).toBeDisabled();
    expect(hover).toHaveAttribute("title", "line 须为 ≥1 的整数");
    fireEvent.change(screen.getByTestId("probe-line"), { target: { value: "-1" } });
    expect(hover).toBeDisabled();
    fireEvent.change(screen.getByTestId("probe-line"), { target: { value: "1.5" } });
    expect(hover).toBeDisabled();
    fireEvent.change(screen.getByTestId("probe-line"), { target: { value: "10" } });
    expect(hover).not.toBeDisabled();
  });

  it("disables actions when char is negative", () => {
    renderProbe();
    const hover = screen.getByTestId("probe-hover");
    fireEvent.change(screen.getByTestId("probe-char"), { target: { value: "-2" } });
    expect(hover).toBeDisabled();
    expect(hover).toHaveAttribute("title", "char 须为 ≥0 的整数");
    fireEvent.change(screen.getByTestId("probe-char"), { target: { value: "0" } });
    expect(hover).not.toBeDisabled();
  });

  it("syncs repoRoot when defaultRepoRoot changes", () => {
    const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const { rerender } = render(
      <QueryClientProvider client={qc}>
        <RagLspProbePanel testIdPrefix="probe" defaultRepoRoot="." />
      </QueryClientProvider>,
    );
    expect(screen.getByTestId("probe-repo-root")).toHaveValue(".");
    rerender(
      <QueryClientProvider client={qc}>
        <RagLspProbePanel testIdPrefix="probe" defaultRepoRoot="../ash" />
      </QueryClientProvider>,
    );
    expect(screen.getByTestId("probe-repo-root")).toHaveValue("../ash");
  });

  it("hovers with the default worker entry path", async () => {
    renderProbe();
    fireEvent.click(screen.getByTestId("probe-hover"));
    await waitFor(() => {
      expect(postRagLspHover).toHaveBeenCalledWith(
        expect.objectContaining({ path: "cmd/worker/main.go", repoRoot: "." }),
      );
    });
  });
});
