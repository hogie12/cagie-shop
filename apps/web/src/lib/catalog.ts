import { cacheLife, cacheTag } from "next/cache";
import type { Product, ProductWithStock } from "./types";

/** Server-side: the product list with stock, cached for about a minute so the
 * home page is fast and part of the prefetched shell. Returns null when the
 * gateway cannot be reached, so the page can say so instead of crashing. */
export async function getCatalog(): Promise<ProductWithStock[] | null> {
  "use cache";
  cacheLife("minutes");
  cacheTag("catalog");

  const base = process.env.API_URL ?? "http://localhost:8080";
  try {
    const res = await fetch(`${base}/products`);
    if (!res.ok) return null;
    const products: Product[] = await res.json();
    return await Promise.all(
      products.map(async (p) => {
        const s = await fetch(`${base}/inventory/${p.id}`);
        const stock = s.ok ? ((await s.json()) as { quantity: number }).quantity : null;
        return { ...p, stock };
      }),
    );
  } catch {
    return null;
  }
}
