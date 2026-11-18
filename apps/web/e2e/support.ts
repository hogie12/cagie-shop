import { readFileSync, existsSync } from "node:fs";
import { join } from "node:path";
import { expect, type APIRequestContext, type Page } from "@playwright/test";

export const GATEWAY = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";
export const STATE_FILE = join(__dirname, ".state", "accounts.json");

export type Accounts = { adminToken: string; customerEmail: string; customerPassword: string };

export function accounts(): Accounts {
  return JSON.parse(readFileSync(STATE_FILE, "utf8"));
}

/** Reads ADMIN_EMAIL / ADMIN_PASSWORD from the repo's root .env (never printed). */
export function adminCredentials(): { email: string; password: string } {
  const file = join(__dirname, "..", "..", "..", ".env");
  if (!existsSync(file)) throw new Error("Root .env not found: run the backend first (make up)");
  const env = Object.fromEntries(
    readFileSync(file, "utf8")
      .split("\n")
      .filter((l) => l.includes("=") && !l.startsWith("#"))
      .map((l) => [l.slice(0, l.indexOf("=")), l.slice(l.indexOf("=") + 1)]),
  );
  return { email: env.ADMIN_EMAIL, password: env.ADMIN_PASSWORD };
}

export async function setProviderMode(api: APIRequestContext, mode: "succeed" | "decline" | "down") {
  const res = await api.put(`${GATEWAY}/payments/provider`, {
    headers: { Authorization: `Bearer ${accounts().adminToken}` },
    data: { mode },
  });
  expect(res.status()).toBe(200);
}

export async function stockOf(api: APIRequestContext, productId: number): Promise<number> {
  const res = await api.get(`${GATEWAY}/inventory/${productId}`);
  return (await res.json()).quantity;
}

export async function setStock(api: APIRequestContext, productId: number, quantity: number) {
  const res = await api.put(`${GATEWAY}/inventory/${productId}`, {
    headers: { Authorization: `Bearer ${accounts().adminToken}` },
    data: { quantity },
  });
  expect(res.status()).toBe(200);
}

export async function signIn(page: Page) {
  const { customerEmail, customerPassword } = accounts();
  await page.goto("/login?next=/cart");
  await page.getByLabel("Email").fill(customerEmail);
  await page.getByLabel("Password").fill(customerPassword);
  await page.getByRole("button", { name: "Sign in", exact: true }).last().click();
  await expect(page).toHaveURL(/\/cart$/);
}

export async function addToCart(page: Page, productName: string) {
  const card = page.getByRole("listitem").filter({ has: page.getByRole("heading", { name: productName }) });
  const stepper = card.getByRole("group", { name: `${productName} in your cart` });
  // Check the stepper first: it appears at once, while the old "Add to cart"
  // button stays in the page for a moment as it fades out.
  if (await stepper.isVisible()) await stepper.getByRole("button", { name: `One more ${productName}` }).click();
  else await card.getByRole("button", { name: "Add to cart" }).click();
  await expect(stepper).toBeVisible();
}
