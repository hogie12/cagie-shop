import { expect, test } from "@playwright/test";
import { addToCart, setProviderMode, setStock, signIn, stockOf } from "./support";

const TART = 3; // Strawberry Tart's product id in the seed data

test.describe("shop", () => {
  test("browsing: real products, photos and the category filter", async ({ page }) => {
    await page.goto("/");
    await expect(page.getByRole("heading", { name: /Pastries in the morning/ })).toBeVisible();
    const products = page.locator("#products");
    await expect(products.getByRole("heading", { name: "Butter Croissant" })).toBeVisible();
    await expect(products.getByRole("img", { name: "Butter Croissant" })).toBeVisible();

    await page.getByRole("button", { name: /Flowers/ }).click();
    await expect(products.getByRole("heading", { name: "Red Rose Bouquet" })).toBeVisible();
    await expect(products.getByRole("heading", { name: "Butter Croissant" })).toHaveCount(0);
  });

  test("a purchase goes through stock and payment and ends confirmed", async ({ page, request }) => {
    await setProviderMode(request, "succeed");
    await page.goto("/");
    await addToCart(page, "Butter Croissant");
    await addToCart(page, "Butter Croissant");
    await addToCart(page, "Sunflower Bunch");
    await expect(page.getByRole("link", { name: /Cart/ })).toContainText("3");

    await signIn(page); // lands on /cart
    await expect(page.getByText("Rp 176.000").first()).toBeVisible(); // 2 × 28.000 + 120.000
    await page.getByRole("button", { name: "Place order" }).click();

    await expect(page).toHaveURL(/\/orders\/[0-9a-f-]{36}$/);
    await expect(page.getByRole("heading", { name: "Order confirmed" })).toBeVisible();
    await expect(page.getByText("thank you")).toBeVisible();

    await page.getByRole("link", { name: "My orders" }).click();
    await expect(page.getByText("Butter Croissant × 2, Sunflower Bunch × 1").first()).toBeVisible();
  });

  test("a declined card cancels the order and puts the stock back", async ({ page, request }) => {
    const before = await stockOf(request, TART);
    await setProviderMode(request, "decline");
    try {
      await page.goto("/");
      await addToCart(page, "Strawberry Tart");
      await signIn(page);
      await page.getByRole("button", { name: "Place order" }).click();

      await expect(page.getByRole("heading", { name: "Order cancelled" })).toBeVisible();
      await expect(page.getByText("Payment failed: card declined. Your items are back on the shelf.")).toBeVisible();
      await expect.poll(() => stockOf(request, TART)).toBe(before); // compensation released it
    } finally {
      await setProviderMode(request, "succeed");
      await setStock(request, TART, before);
    }
  });
});
