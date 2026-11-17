import { describe, expect, it } from "vitest";
import { addLine, cartTotals, MAX_LINES, MAX_QUANTITY, setLineQuantity, type CartLine } from "./cart";
import { rupiah, stockLabel } from "./format";
import { illustrationFor } from "./illustrations";
import { decodeClaims } from "./jwt";
import { describeOrder, friendlyReason } from "./orderStatus";
import { safeNext } from "../app/login/LoginForm";

const croissant = { productId: 1, name: "Butter Croissant", category: "pastry" as const, priceIdr: 28000, imageUrl: "" };
const roses = { productId: 4, name: "Red Rose Bouquet", category: "flower" as const, priceIdr: 350000, imageUrl: "" };

describe("rupiah", () => {
  it("uses dots as thousands separators", () => {
    expect(rupiah(28000)).toBe("Rp 28.000");
    expect(rupiah(1250000)).toBe("Rp 1.250.000");
    expect(rupiah(0)).toBe("Rp 0");
    expect(rupiah(999)).toBe("Rp 999");
  });
});

describe("stockLabel", () => {
  it("warns when stock is low and handles missing records", () => {
    expect(stockLabel(20)).toBe("20 in stock");
    expect(stockLabel(3)).toBe("Only 3 left");
    expect(stockLabel(0)).toBe("Sold out");
    expect(stockLabel(null)).toBe("Not stocked yet");
  });
});

describe("cart", () => {
  it("adds, merges repeated products and totals", () => {
    let lines: CartLine[] = [];
    lines = addLine(lines, croissant, 2);
    lines = addLine(lines, roses);
    lines = addLine(lines, croissant);
    expect(lines).toHaveLength(2);
    expect(lines[0].quantity).toBe(3);
    expect(cartTotals(lines)).toEqual({ count: 4, totalIdr: 3 * 28000 + 350000 });
  });

  it("keeps quantities within the order service's limits", () => {
    let lines = addLine([], croissant, 98);
    lines = addLine(lines, croissant, 5);
    expect(lines[0].quantity).toBe(MAX_QUANTITY);
    let many: CartLine[] = [];
    for (let i = 1; i <= MAX_LINES + 3; i++) many = addLine(many, { ...croissant, productId: i });
    expect(many).toHaveLength(MAX_LINES);
  });

  it("removes a line when its quantity drops to zero", () => {
    const lines = addLine(addLine([], croissant), roses);
    expect(setLineQuantity(lines, 1, 0).map((l) => l.productId)).toEqual([4]);
    expect(setLineQuantity(lines, 4, 7).find((l) => l.productId === 4)?.quantity).toBe(7);
  });
});

describe("describeOrder", () => {
  it("follows the saga step by step", () => {
    expect(describeOrder({ status: "pending" }).steps.map((s) => s.state)).toEqual(["done", "current", "todo", "todo"]);
    expect(describeOrder({ status: "awaiting_payment" }).steps.map((s) => s.state)).toEqual(["done", "done", "current", "todo"]);
    const confirmed = describeOrder({ status: "confirmed" });
    expect(confirmed.steps.every((s) => s.state === "done")).toBe(true);
    expect(confirmed.live).toBe(false);
  });

  it("shows where a cancelled order failed", () => {
    const stock = describeOrder({ status: "cancelled", cancelReason: "insufficient stock for product 4" });
    expect(stock.failedAt).toBe("stock");
    expect(stock.steps[1].state).toBe("failed");
    const payment = describeOrder({ status: "cancelled", cancelReason: "payment failed: card declined" });
    expect(payment.failedAt).toBe("payment");
    expect(payment.steps.map((s) => s.state)).toEqual(["done", "done", "failed", "todo"]);
  });

  it("rewrites backend reasons for customers", () => {
    expect(friendlyReason("payment failed: card declined")).toBe("Payment failed: card declined. Your items are back on the shelf.");
    expect(friendlyReason("insufficient stock for product 4")).toMatch(/Not enough stock/);
    expect(friendlyReason(undefined)).toMatch(/could not be completed/);
  });
});

describe("illustrationFor", () => {
  it("matches product names, then falls back by category", () => {
    expect(illustrationFor("Chocolate Eclair", "pastry")).toBe("eclair");
    expect(illustrationFor("White Lily Vase", "flower")).toBe("lily");
    expect(illustrationFor("Cinnamon Roll", "pastry")).toBe("croissant");
    expect(illustrationFor("Tulip Bunch", "flower")).toBe("sprig");
  });
});

describe("decodeClaims", () => {
  const encode = (o: object) => Buffer.from(JSON.stringify(o)).toString("base64url");
  it("reads the user id and role from an access token", () => {
    const token = `${encode({ alg: "RS256" })}.${encode({ sub: "u1", role: "admin", exp: 9999999999 })}.sig`;
    expect(decodeClaims(token)).toEqual({ sub: "u1", role: "admin", exp: 9999999999 });
  });
  it("rejects garbage and unknown roles", () => {
    expect(decodeClaims("nope")).toBeNull();
    expect(decodeClaims(`x.${encode({ sub: "u1", role: "root", exp: 1 })}.y`)).toBeNull();
  });
});

describe("safeNext", () => {
  it("only allows paths on this site", () => {
    expect(safeNext("/cart")).toBe("/cart");
    expect(safeNext("//evil.example")).toBe("/");
    expect(safeNext("/\\evil.example")).toBe("/");
    expect(safeNext("https://evil.example")).toBe("/");
    expect(safeNext(null)).toBe("/");
  });
});
