import { test, expect } from "@playwright/test";

test.describe("Navigation", () => {
  test("brand logo navigates to homepage", async ({ page }) => {
    await page.goto("/portfolio");
    await page.locator("text=fo").first().click();
    await expect(page).toHaveURL("/");
  });

  test("clicking a market card navigates to detail page", async ({ page }) => {
    await page.goto("/");
    const card = page.locator("a[href^='/markets/']").first();
    const exists = (await card.count()) > 0;

    if (exists) {
      const href = await card.getAttribute("href");
      await card.click();
      await expect(page).toHaveURL(new RegExp(`/markets/\\d+`));

      const backButton = page.locator("text=fo").first();
      await expect(backButton).toBeVisible();
    }
  });

  test("portfolio page loads", async ({ page }) => {
    await page.goto("/portfolio");
    await expect(page).toHaveURL("/portfolio");
  });

  test("control page loads", async ({ page }) => {
    await page.goto("/control");
    await expect(page).toHaveURL("/control");
  });

  test("invalid market ID shows error page", async ({ page }) => {
    await page.goto("/markets/0");
    const errorHeading = page.getByRole("heading", { name: /不是有效/ });
    await expect(errorHeading).toBeVisible();
  });
});
