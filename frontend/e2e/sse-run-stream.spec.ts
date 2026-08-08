import { expect, test } from "@playwright/test";

/**
 * Browser-level SSE smoke (P2-4): open Runs console against a live Worker,
 * select a run, assert EventSource reaches "已连接" and delivers ≥1 event line.
 */
test.describe("SSE run stream (browser)", () => {
  test("select run → SSE connected with events", async ({ page, request }) => {
    const create = await request.post("/api/v1/runs", {
      headers: {
        "Content-Type": "application/json",
        "X-ASH-Space-ID": "local",
      },
      data: {
        scenario: { name: "feature_delivery", scenarioVersion: "1.0.0" },
        actorRole: "maintainer",
        inputs: {
          issueOrSpec: `playwright sse ${Date.now()}`,
          repoRoot: ".",
        },
      },
    });
    expect(create.ok(), await create.text()).toBeTruthy();
    const body = (await create.json()) as { runId: string };
    expect(body.runId).toBeTruthy();

    await page.goto("/ui/runs");
    await expect(page.getByRole("heading", { name: "运行", exact: true })).toBeVisible();
    await expect(page.getByRole("heading", { name: "事件流 (SSE)" })).toBeVisible();

    // Prefer clicking the queue row that contains the new run id (short or full).
    const row = page.locator("tr", { hasText: body.runId.slice(0, 12) }).first();
    await expect(row).toBeVisible({ timeout: 30_000 });
    await row.click();

    const status = page.getByTestId("sse-stream-status");
    await expect(status).toHaveAttribute("data-stream-status", "open", { timeout: 45_000 });
    await expect(status).toContainText("已连接");

    await expect(page.getByTestId("sse-event-line").first()).toBeVisible({ timeout: 45_000 });
    await expect(status).toContainText(/[1-9]\d*\s*条事件/);
  });
});
