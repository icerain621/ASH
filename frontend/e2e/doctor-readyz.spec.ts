import { expect, test } from "@playwright/test";

/**
 * Doctor / readyz smoke: console Doctor page + Worker health endpoints.
 * Does not auto-run Doctor suites (read-only UI + API probes).
 */
test.describe("doctor + readyz", () => {
  test("readyz and healthz are OK", async ({ request }) => {
    const ready = await request.get("/readyz");
    expect(ready.ok(), await ready.text()).toBeTruthy();
    const health = await request.get("/healthz");
    expect(health.ok(), await health.text()).toBeTruthy();
  });

  test("Doctor page renders suite controls", async ({ page }) => {
    await page.goto("/ui/doctor");
    await expect(page.getByRole("heading", { name: "诊断", level: 1 })).toBeVisible({
      timeout: 30_000,
    });
    // Suite picker / run controls exist without executing a suite.
    const body = page.locator("body");
    await expect(body).toContainText(/TR0|TR3|ALL|M2|M3|套件|诊断/i);
  });
});
