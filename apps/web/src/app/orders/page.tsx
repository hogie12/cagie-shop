"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { motion } from "motion/react";
import { Illustration } from "@/components/Illustration";
import { SignInPrompt } from "@/components/SignInPrompt";
import { StatusPill } from "@/components/StatusPill";
import { useAuth } from "@/lib/auth";
import { orderDate, rupiah, shortId } from "@/lib/format";
import type { Order } from "@/lib/types";

export default function OrdersPage() {
  const { status, authRequest } = useAuth();
  const [orders, setOrders] = useState<Order[] | null>(null);
  const [error, setError] = useState(false);

  useEffect(() => {
    if (status !== "signedIn") return;
    authRequest<Order[]>("/orders").then(setOrders, () => setError(true));
  }, [status, authRequest]);

  return (
    <main className="mx-auto w-full max-w-[880px] px-6 pb-16 pt-8">
      <p className="font-hand text-[26px] text-rose">your history</p>
      <h1 className="mb-7 font-serif text-[44px] font-bold">My orders</h1>

      {status === "signedOut" && <SignInPrompt next="/orders" text="Sign in to see your orders." />}
      {status !== "signedOut" && error && <p role="alert" className="text-rose-deep">Your orders could not be loaded. Please refresh the page.</p>}
      {status !== "signedOut" && !error && orders === null && <p className="text-muted">Loading your orders…</p>}
      {orders?.length === 0 && (
        <div className="flex flex-col items-center gap-3 rounded-3xl border border-line bg-white p-12 text-center">
          <Illustration kind="sunflower" size={88} decorative sway />
          <p className="text-muted">No orders yet.</p>
          <Link href="/#products" className="inline-flex min-h-11 items-center rounded-full bg-forest px-5 font-semibold text-white">
            Browse the shop
          </Link>
        </div>
      )}
      {orders && orders.length > 0 && (
        <ul className="flex list-none flex-col gap-3 p-0">
          {orders.map((o, i) => (
            <motion.li key={o.id} initial={{ opacity: 0, y: 12 }} animate={{ opacity: 1, y: 0 }} transition={{ delay: i * 0.04 }}>
              <Link href={`/orders/${o.id}`} className="flex flex-wrap items-center justify-between gap-3 rounded-[20px] border border-line bg-white p-5 text-ink transition-colors hover:border-forest">
                <span className="flex flex-col">
                  <span className="font-semibold">
                    Order <span className="font-mono">{shortId(o.id)}</span>
                  </span>
                  <span className="text-sm text-muted">
                    {orderDate(o.createdAt)} · {o.items.map((it) => `${it.name} × ${it.quantity}`).join(", ")}
                  </span>
                </span>
                <span className="flex items-center gap-4">
                  <StatusPill status={o.status} />
                  <span className="font-bold tabular-nums">{rupiah(o.totalIdr)}</span>
                </span>
              </Link>
            </motion.li>
          ))}
        </ul>
      )}
    </main>
  );
}
