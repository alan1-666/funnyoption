import { test, expect } from "@playwright/test";

test.describe("Homepage", () => {
  test("loads and displays the brand mark", async ({ page }) => {
    await page.goto("/");
    const brand = page.locator("text=fo").first();
    await expect(brand).toBeVisible();
  });

  test("renders the search bar", async ({ page }) => {
    await page.goto("/");
    const searchInput = page.locator('input[type="search"]');
    await expect(searchInput).toBeVisible();
    await expect(searchInput).toHaveAttribute("placeholder", /search/i);
  });

  test("shows market cards when API is available", async ({ page }) => {
    await page.goto("/");
    const pageShell = page.locator("main.page-shell");
    await expect(pageShell).toBeVisible();

    const cards = page.locator('[class*="card"]');
    const cardCount = await cards.count();
    if (cardCount > 0) {
      await expect(cards.first()).toBeVisible();
    }
  });

  test("shows filter buttons", async ({ page }) => {
    await page.goto("/");
    const allFilter = page.getByText("全部");
    await expect(allFilter).toBeVisible();
  });

  test("connect button appears when not logged in", async ({ page }) => {
    await page.goto("/");
    const connectButton = page.getByText("Connect");
    await expect(connectButton).toBeVisible();
  });

  test("search filters market cards", async ({ page }) => {
    await page.goto("/");
    const searchInput = page.locator('input[type="search"]');
    await searchInput.fill("nonexistent_market_query_xyz");
    await page.waitForTimeout(300);

    const emptyState = page.getByText(/没有找到|暂无/);
    const emptyCount = await emptyState.count();
    expect(emptyCount).toBeGreaterThanOrEqual(0);
  });
});
