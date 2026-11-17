"use client";

import { useSyncExternalStore } from "react";
import type { Category } from "./types";

export type CartLine = {
  productId: number;
  name: string;
  category: Category;
  priceIdr: number;
  imageUrl: string;
  quantity: number;
};

// Same limits as the order service, so a cart can always be submitted.
export const MAX_QUANTITY = 99;
export const MAX_LINES = 20;

// ---- Pure functions (unit tested) ----

export function addLine(lines: CartLine[], item: Omit<CartLine, "quantity">, quantity = 1): CartLine[] {
  const existing = lines.find((l) => l.productId === item.productId);
  if (existing) {
    return lines.map((l) => (l.productId === item.productId ? { ...l, quantity: Math.min(l.quantity + quantity, MAX_QUANTITY) } : l));
  }
  if (lines.length >= MAX_LINES) return lines;
  return [...lines, { ...item, quantity: Math.min(Math.max(quantity, 1), MAX_QUANTITY) }];
}

export function setLineQuantity(lines: CartLine[], productId: number, quantity: number): CartLine[] {
  if (quantity <= 0) return lines.filter((l) => l.productId !== productId);
  return lines.map((l) => (l.productId === productId ? { ...l, quantity: Math.min(quantity, MAX_QUANTITY) } : l));
}

export function cartTotals(lines: CartLine[]): { count: number; totalIdr: number } {
  return lines.reduce((t, l) => ({ count: t.count + l.quantity, totalIdr: t.totalIdr + l.quantity * l.priceIdr }), { count: 0, totalIdr: 0 });
}

// ---- Browser store (localStorage) ----
// Prices here are only for display: the order service takes prices from the
// catalog when the order is placed.

const KEY = "cagie-cart";
const EMPTY: CartLine[] = [];
const listeners = new Set<() => void>();
let cache: CartLine[] | null = null;

function read(): CartLine[] {
  if (cache) return cache;
  try {
    const parsed = JSON.parse(localStorage.getItem(KEY) ?? "[]");
    cache = Array.isArray(parsed) ? parsed : EMPTY;
  } catch {
    cache = EMPTY;
  }
  return cache;
}

function write(lines: CartLine[]) {
  cache = lines;
  try {
    localStorage.setItem(KEY, JSON.stringify(lines));
  } catch {
    // Private mode or full storage: the cart still works for this visit.
  }
  listeners.forEach((l) => l());
}

function subscribe(listener: () => void) {
  listeners.add(listener);
  const onStorage = (e: StorageEvent) => {
    if (e.key === KEY) {
      cache = null;
      listener();
    }
  };
  window.addEventListener("storage", onStorage); // keeps other tabs in sync
  return () => {
    listeners.delete(listener);
    window.removeEventListener("storage", onStorage);
  };
}

export const cart = {
  add: (item: Omit<CartLine, "quantity">, quantity = 1) => write(addLine(read(), item, quantity)),
  setQuantity: (productId: number, quantity: number) => write(setLineQuantity(read(), productId, quantity)),
  clear: () => write(EMPTY),
};

/** The cart lines; empty during server rendering, filled in after hydration. */
export function useCart(): CartLine[] {
  return useSyncExternalStore(subscribe, read, () => EMPTY);
}
