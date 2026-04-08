import { test, expect } from "@playwright/test";

test.describe("Notification system", () => {
  test("notification bell is visible in top bar", async ({ page }) => {
    await page.goto("/");
    const bellButton = page.locator('button[aria-label="Notifications"]');
    await expect(bellButton).toBeVisible();
  });

  test("notification bell is disabled when not logged in", async ({ page }) => {
    await page.goto("/");
    const bellButton = page.locator('button[aria-label="Notifications"]');
    await expect(bellButton).toBeDisabled();
  });
});
