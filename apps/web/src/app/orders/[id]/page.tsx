import { Suspense } from "react";
import type { Metadata } from "next";
import { OrderDetail } from "./OrderDetail";

export const metadata: Metadata = { title: "Your order" };

// The order id comes from the URL, which this Next version only lets a page
// read inside a Suspense boundary.
export default function OrderPage() {
  return (
    <Suspense fallback={<main className="mx-auto w-full max-w-[880px] px-6 pt-8 text-muted">Loading your order…</main>}>
      <OrderDetail />
    </Suspense>
  );
}
