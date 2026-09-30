import { expect, test } from "@playwright/test";

/**
 * Memory console smoke: list / tabs render; create-candidate path is not submitted.
 */
test.describe("memory console", () => {
  test("memory page shows tabs and candidate form shell", async ({ page }) => {
    await page.goto("/ui/memory");
    await expect(page.getByRole("heading", { name: "记忆", level: 1 })).toBeVisible({
      timeout: 30_000,
    });
    await expect(page.getByRole("button", { name: /^候选$|^列表$|候选/ }).first()).toBeVisible();
    // Association / links tab is navigable without mutating memory.
    const linksTab = page.getByTestId("memory-tab-links");
    if ((await linksTab.count()) > 0) {
      await linksTab.click();
      await expect(page.locator("body")).toContainText(/关联|链接|run/i);
    }
  });

  test("memory list API responds for local space", async ({ request }) => {
    const res = await request.get("/api/v1/memory/candidates?limit=5", {
      headers: { "X-ASH-Space-ID": "local" },
    });
    expect(res.ok(), await res.text()).toBeTruthy();
    const body = (await res.json()) as { items?: unknown[] };
    expect(Array.isArray(body.items)).toBeTruthy();
  });
});
