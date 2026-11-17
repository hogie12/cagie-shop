import type { Category } from "./types";

export type IllustrationKind = "croissant" | "eclair" | "tart" | "rose" | "sunflower" | "lily" | "sprig";

const byKeyword: [string, IllustrationKind][] = [
  ["croissant", "croissant"],
  ["eclair", "eclair"],
  ["tart", "tart"],
  ["rose", "rose"],
  ["sunflower", "sunflower"],
  ["lily", "lily"],
];

/** Picks a drawing for a product: by name first, then by category. */
export function illustrationFor(name: string, category: Category): IllustrationKind {
  const lower = name.toLowerCase();
  for (const [word, kind] of byKeyword) {
    if (lower.includes(word)) return kind;
  }
  return category === "pastry" ? "croissant" : "sprig";
}
