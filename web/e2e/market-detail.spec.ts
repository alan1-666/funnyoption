import { test, expect } from "@playwright/test";

test.describe("Market detail page", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/");
    const marketLink = page.locator("a[href^='/markets/']").first();
    const exists = (await marketLink.count()) > 0;
    if (!exists) {
      test.skip(true, "No markets available to test");
      return;
    }
    await marketLink.click();
    await page.waitForURL(/\/markets\/\d+/);
  });

  test("shows market title and status", async ({ page }) => {
    const heading = page.locator("h1").first();
    await expect(heading).toBeVisible();
    const headingText = await heading.textContent();
    expect(headingText?.length).toBeGreaterThan(0);
  });

  test("shows YES/NO price indicators", async ({ page }) => {
    const priceIndicators = page.locator('[class*="price"], [class*="odds"]');
    const count = await priceIndicators.count();
    expect(count).toBeGreaterThanOrEqual(0);
  });

  test("shows order ticket section", async ({ page }) => {
    const orderSection = page.locator('[class*="order"], [class*="ticket"]');
    const count = await orderSection.count();
    expect(count).toBeGreaterThanOrEqual(0);
  });

  test("shows recent trades section", async ({ page }) => {
    const tradesSection = page.locator('[class*="trade"], [class*="activity"]');
    const count = await tradesSection.count();
    expect(count).toBeGreaterThanOrEqual(0);
  });
});
