import { test, expect } from "@playwright/test";

test.describe("Market creation (propose form)", () => {
  test("propose button is disabled when not logged in", async ({ page }) => {
    await page.goto("/");
    const proposeButton = page.locator('button[aria-label="Propose a market"]');
    await expect(proposeButton).toBeVisible();
    await expect(proposeButton).toBeDisabled();
  });

  test("propose button exists in top bar", async ({ page }) => {
    await page.goto("/");
    const svg = page.locator('button[aria-label="Propose a market"] svg');
    await expect(svg).toBeVisible();
  });
});
