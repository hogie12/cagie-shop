"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { AnimatePresence, motion } from "motion/react";
import { Illustration } from "@/components/Illustration";
import { ProductThumb } from "@/components/ProductThumb";
import { ApiError } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { cart, cartTotals, MAX_QUANTITY, useCart } from "@/lib/cart";
import { rupiah } from "@/lib/format";
import type { Order } from "@/lib/types";

function orderError(err: unknown): string {
  if (!(err instanceof ApiError)) return "Something went wrong. Please try again.";
  if (err.status === 0) return err.message;
  if (err.status === 422) return "One of your items is no longer sold. Remove it and try again.";
  if (err.status === 400) return "Your cart has an invalid quantity. Each item can be 1 to 99.";
  if (err.status === 502) return "The shop is busy for a moment. Please try again.";
  return "Your order could not be placed. Please try again.";
}

export default function CartPage() {
  const lines = useCart();
  const { count, totalIdr } = cartTotals(lines);
  const { status, user, authRequest } = useAuth();
  const router = useRouter();
  const [placing, setPlacing] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const placeOrder = async () => {
    setPlacing(true);
    setError(null);
    try {
      const order = await authRequest<Order>("/orders", {
        method: "POST",
        body: { items: lines.map((l) => ({ productId: l.productId, quantity: l.quantity })) },
      });
      cart.clear();
      router.push(`/orders/${order.id}`);
    } catch (err) {
      setError(orderError(err));
      setPlacing(false);
    }
  };

  return (
    <main className="mx-auto w-full max-w-[1180px] px-6 pb-16 pt-8">
      <Link href="/" className="inline-flex items-center gap-1.5 py-2.5 text-sm font-semibold text-forest">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
          <path d="M15 18l-6-6 6-6" />
        </svg>
        Keep shopping
      </Link>
      <h1 className="mb-7 mt-2 font-serif text-[44px] font-bold tracking-[-0.01em]">Your cart</h1>

      {lines.length === 0 ? (
        <div className="flex flex-col items-center gap-3 rounded-3xl border border-line bg-white p-12 text-center">
          <Illustration kind="croissant" size={96} decorative />
          <h2 className="font-serif text-2xl font-bold">Your cart is empty</h2>
          <p className="text-muted">Pick a pastry or a bouquet to get started.</p>
          <Link href="/#products" className="mt-2 inline-flex min-h-11 items-center rounded-full bg-forest px-5 font-semibold text-white">
            Browse the shop
          </Link>
        </div>
      ) : (
        <div className="flex flex-wrap items-start gap-6">
          <section aria-label="Items" className="min-w-0 flex-[999_1_560px] rounded-[22px] border border-line bg-white">
            <ul className="list-none p-0">
              <AnimatePresence initial={false}>
                {lines.map((l) => (
                  <motion.li
                    key={l.productId}
                    layout
                    exit={{ opacity: 0, height: 0, transition: { duration: 0.25 } }}
                    className="flex flex-wrap items-center gap-4 border-b border-line p-5 last:border-b-0"
                  >
                    <ProductThumb name={l.name} category={l.category} imageUrl={l.imageUrl} size={76} />
                    <div className="min-w-0 flex-[1_1_200px]">
                      <p className="font-serif text-[19px] font-bold">{l.name}</p>
                      <p className="mt-0.5 text-sm text-muted">{rupiah(l.priceIdr)} each</p>
                    </div>
                    <div className="inline-flex items-center rounded-full border border-line bg-ground">
                      <button type="button" aria-label={`One less ${l.name}`} onClick={() => cart.setQuantity(l.productId, l.quantity - 1)} className="h-11 w-11 cursor-pointer text-lg">
                        −
                      </button>
                      <motion.span key={l.quantity} initial={{ scale: 1.3 }} animate={{ scale: 1 }} className="min-w-7 text-center font-semibold tabular-nums">
                        {l.quantity}
                      </motion.span>
                      <button
                        type="button"
                        aria-label={`One more ${l.name}`}
                        disabled={l.quantity >= MAX_QUANTITY}
                        onClick={() => cart.setQuantity(l.productId, l.quantity + 1)}
                        className="h-11 w-11 cursor-pointer text-lg disabled:opacity-40"
                      >
                        +
                      </button>
                    </div>
                    <p className="w-[120px] text-right font-bold tabular-nums">{rupiah(l.priceIdr * l.quantity)}</p>
                  </motion.li>
                ))}
              </AnimatePresence>
            </ul>
          </section>

          <aside aria-label="Order summary" className="relative flex min-w-0 flex-[1_1_340px] flex-col gap-4 overflow-hidden rounded-[22px] border border-line bg-white p-6">
            <div className="absolute -right-4 -top-4 opacity-90">
              <Illustration kind="sprig" size={96} decorative />
            </div>
            <h2 className="font-serif text-2xl font-bold">Summary</h2>
            <div className="flex justify-between text-muted">
              <span>
                {count} {count === 1 ? "item" : "items"}
              </span>
              <span className="tabular-nums">{rupiah(totalIdr)}</span>
            </div>
            <div className="flex justify-between border-t border-line pt-4 text-lg font-bold">
              <span>Total</span>
              <span className="tabular-nums">{rupiah(totalIdr)}</span>
            </div>

            {status === "signedIn" && user ? (
              <motion.button
                type="button"
                whileTap={{ scale: 0.97 }}
                disabled={placing}
                onClick={() => void placeOrder()}
                className="min-h-[52px] cursor-pointer rounded-full bg-forest text-base font-semibold text-white hover:bg-forest-dark disabled:opacity-70"
              >
                {placing ? "Placing your order…" : "Place order"}
              </motion.button>
            ) : (
              <Link href="/login?next=/cart" className="flex min-h-[52px] items-center justify-center rounded-full bg-forest font-semibold text-white">
                {status === "loading" ? "Checking your account…" : "Sign in to place your order"}
              </Link>
            )}

            {error && (
              <p role="alert" className="rounded-xl bg-rose-tint p-3 text-sm text-rose-deep">
                {error}
              </p>
            )}
            <div className="flex gap-2.5 rounded-[14px] bg-forest-tint p-3.5 text-sm text-forest">
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true" className="mt-0.5 shrink-0">
                <circle cx="12" cy="12" r="9" />
                <path d="M12 8v4" />
                <path d="M12 16h.01" />
              </svg>
              <span>After you place the order, Cagie reserves your items and takes payment. You will see each step on the next page.</span>
            </div>
          </aside>
        </div>
      )}
    </main>
  );
}
