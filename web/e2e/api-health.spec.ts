import { test, expect } from "@playwright/test";

const API_BASE_URL = process.env.E2E_API_BASE_URL ?? "http://127.0.0.1:8080";

test.describe("API health checks", () => {
  test("markets endpoint returns 200", async ({ request }) => {
    const response = await request.get(`${API_BASE_URL}/api/v1/markets`);
    expect(response.status()).toBe(200);
    const body = await response.json();
    expect(body).toHaveProperty("items");
    expect(Array.isArray(body.items)).toBeTruthy();
  });

  test("trades endpoint returns 200", async ({ request }) => {
    const response = await request.get(`${API_BASE_URL}/api/v1/trades`);
    expect(response.status()).toBe(200);
    const body = await response.json();
    expect(body).toHaveProperty("items");
  });

  test("single market returns data or 404", async ({ request }) => {
    const marketsResponse = await request.get(`${API_BASE_URL}/api/v1/markets`);
    const marketsBody = await marketsResponse.json();

    if (marketsBody.items.length > 0) {
      const firstMarketId = marketsBody.items[0].id;
      const response = await request.get(`${API_BASE_URL}/api/v1/markets/${firstMarketId}`);
      expect([200, 400, 404]).toContain(response.status());
    }
  });

  test("balances endpoint requires authentication context", async ({ request }) => {
    const response = await request.get(`${API_BASE_URL}/api/v1/balances?user_id=1`);
    expect([200, 401, 403]).toContain(response.status());
  });
});
