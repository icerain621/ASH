import { expect, test, type Page } from "@playwright/test";

/**
 * Console navigation smoke: key /ui routes render their primary headings
 * against a live Worker that serves frontend/dist (see scripts/console-browser-e2e.sh).
 * Read-only — does not create runs or mutate production-like state beyond page loads.
 */

async function expectHeading(page: Page, name: string | RegExp) {
  await expect(page.getByRole("heading", { name, level: 1 })).toBeVisible({ timeout: 30_000 });
}

const routes: Array<{ path: string; heading: string | RegExp }> = [
  { path: "/ui/runs", heading: "运行" },
  { path: "/ui/memory", heading: "记忆" },
  { path: "/ui/quest", heading: "Agent" },
  { path: "/ui/doctor", heading: "诊断" },
  { path: "/ui/space", heading: "空间" },
  { path: "/ui/reviews", heading: "评审管控" },
  { path: "/ui/observability", heading: "可观测与告警" },
  { path: "/ui/metrics", heading: "指标看板" },
  { path: "/ui/ci", heading: "CI 诊断控制台" },
  { path: "/ui/releases", heading: "发布与灰度回滚" },
  { path: "/ui/automation", heading: "自动化" },
  { path: "/ui/feedback", heading: "反馈闭环" },
  { path: "/ui/compliance", heading: "合规控制台" },
  { path: "/ui/scale", heading: "规模化就绪" },
  { path: "/ui/login", heading: "登录" },
];

test.describe("console navigation smoke", () => {
  for (const route of routes) {
    test(`loads ${route.path}`, async ({ page }) => {
      await page.goto(route.path);
      await expectHeading(page, route.heading);
    });
  }

  test("unknown path shows not-found", async ({ page }) => {
    await page.goto("/ui/this-route-does-not-exist-xyz");
    await expectHeading(page, "页面不存在");
    await expect(page.getByTestId("console-not-found-home")).toBeVisible();
  });
});
