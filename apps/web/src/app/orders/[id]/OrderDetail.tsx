"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useEffect, useState } from "react";
import { AnimatePresence, motion } from "motion/react";
import { Illustration } from "@/components/Illustration";
import { OrderTracker } from "@/components/OrderTracker";
import { SignInPrompt } from "@/components/SignInPrompt";
import { ApiError } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { orderDate, rupiah, shortId } from "@/lib/format";
import { describeOrder, friendlyReason } from "@/lib/orderStatus";
import type { Order } from "@/lib/types";

const POLL_MS = 1000;
// The saga normally settles in seconds; a payment outage can take ~15s of retries.
const GIVE_UP_MS = 2 * 60 * 1000;

export function OrderDetail() {
  const { id } = useParams<{ id: string }>();
  const { status, authRequest } = useAuth();
  const [order, setOrder] = useState<Order | null>(null);
  const [problem, setProblem] = useState<"notFound" | "error" | "slow" | null>(null);

  // Poll every second until the order is confirmed or cancelled.
  useEffect(() => {
    if (status !== "signedIn") return;
    let stopped = false;
    let timer: ReturnType<typeof setTimeout>;
    const started = Date.now();

    const tick = async () => {
      try {
        const o = await authRequest<Order>(`/orders/${id}`);
        if (stopped) return;
        setOrder(o);
        if (o.status === "confirmed" || o.status === "cancelled") return;
        if (Date.now() - started > GIVE_UP_MS) {
          setProblem("slow");
          return;
        }
      } catch (err) {
        if (stopped) return;
        if (err instanceof ApiError && err.status === 404) {
          setProblem("notFound");
          return;
        }
        setProblem("error"); // network blip: keep trying
      }
      timer = setTimeout(tick, POLL_MS);
    };
    void tick();
    return () => {
      stopped = true;
      clearTimeout(timer);
    };
  }, [id, status, authRequest]);

  if (status === "signedOut") {
    return (
      <main className="mx-auto w-full max-w-[880px] px-6 pb-16 pt-8">
        <SignInPrompt next={`/orders/${id}`} text="Sign in to see this order." />
      </main>
    );
  }
  if (problem === "notFound") {
    return (
      <main className="mx-auto w-full max-w-[880px] px-6 pb-16 pt-8">
        <h1 className="font-serif text-4xl font-bold">Order not found</h1>
        <p className="mt-2 text-muted">This order does not exist or belongs to another account.</p>
        <Link href="/orders" className="mt-4 inline-block font-semibold">
          See my orders
        </Link>
      </main>
    );
  }
  if (!order) return <main className="mx-auto w-full max-w-[880px] px-6 pt-8 text-muted">Loading your order…</main>;

  const view = describeOrder(order);
  const art = view.confirmed ? (["rose", "croissant"] as const) : view.cancelled ? (["sprig", "lily"] as const) : (["croissant", "sprig"] as const);

  return (
    <main className="mx-auto flex w-full max-w-[880px] flex-col gap-6 px-6 pb-16 pt-8">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <p className="text-sm text-muted">
          Order <span className="font-mono text-ink">{shortId(order.id)}</span> · placed {orderDate(order.createdAt)}
        </p>
        {view.live && (
          <span className="inline-flex items-center gap-2 rounded-full bg-forest-tint px-3 py-1.5 text-[13px] font-semibold text-forest">
            <span aria-hidden="true" className="h-2 w-2 rounded-full bg-forest" />
            Updating automatically
          </span>
        )}
      </div>

      <section className="relative flex flex-col gap-7 overflow-hidden rounded-3xl border border-line bg-white p-8">
        <div className="absolute right-6 top-5 hidden items-end sm:flex" aria-hidden="true">
          <AnimatePresence mode="wait">
            <motion.div
              key={art.join()}
              className="flex items-end"
              initial={{ opacity: 0, y: 20, scale: 0.8 }}
              animate={{ opacity: 1, y: 0, scale: 1 }}
              exit={{ opacity: 0, y: -10 }}
              transition={{ type: "spring", stiffness: 200, damping: 16 }}
            >
              <Illustration kind={art[0]} size={96} decorative />
              <Illustration kind={art[1]} size={80} decorative />
            </motion.div>
          </AnimatePresence>
        </div>

        {/* Screen readers hear each status change. */}
        <div aria-live="polite" className="sm:pr-48">
          <AnimatePresence mode="wait" initial={false}>
            <motion.div key={order.status} initial={{ opacity: 0, y: 10 }} animate={{ opacity: 1, y: 0 }} exit={{ opacity: 0, y: -10 }} transition={{ duration: 0.25 }}>
              <h1 className="font-serif text-[44px] font-bold tracking-[-0.01em]">{view.headline}</h1>
              {view.confirmed && <p className="font-hand text-3xl text-rose">thank you</p>}
              <p className="mt-1.5 text-base text-muted">{view.subline}</p>
            </motion.div>
          </AnimatePresence>
        </div>

        <OrderTracker steps={view.steps} />

        <AnimatePresence>
          {view.cancelled && (
            <motion.div role="status" initial={{ opacity: 0, height: 0 }} animate={{ opacity: 1, height: "auto" }} className="flex gap-3 rounded-[14px] bg-rose-tint p-4 text-rose-deep">
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true" className="mt-0.5 shrink-0">
                <circle cx="12" cy="12" r="9" />
                <path d="M12 8v4" />
                <path d="M12 16h.01" />
              </svg>
              <p className="font-semibold">{friendlyReason(order.cancelReason)}</p>
            </motion.div>
          )}
        </AnimatePresence>
        {problem === "slow" && <p className="rounded-xl bg-sun p-3 text-sm">This is taking longer than usual. Refresh the page in a little while to see the result.</p>}
        {problem === "error" && view.live && <p className="text-sm text-muted">Reconnecting…</p>}
      </section>

      <section aria-label="Items" className="rounded-3xl border border-line bg-white px-6 py-2">
        {order.items.map((it) => (
          <div key={it.productId} className="flex items-center justify-between gap-4 border-b border-line py-3.5">
            <span>
              {it.name} <span className="text-muted">× {it.quantity}</span>
            </span>
            <span className="tabular-nums">{rupiah(it.unitPriceIdr * it.quantity)}</span>
          </div>
        ))}
        <div className="flex justify-between gap-4 py-4 text-[17px] font-bold">
          <span>Total</span>
          <span className="tabular-nums">{rupiah(order.totalIdr)}</span>
        </div>
      </section>

      <Link href="/" className="inline-flex min-h-11 items-center self-start rounded-full border-[1.5px] border-forest px-[18px] font-semibold text-forest">
        Back to the shop
      </Link>
    </main>
  );
}
