"use client";

import Image from "next/image";
import Link from "next/link";
import { motion } from "motion/react";
import { cart, useCart } from "@/lib/cart";
import { rupiah } from "@/lib/format";
import type { ProductWithStock } from "@/lib/types";
import { Illustration } from "../Illustration";

export function GiftPairing({ flower, pastry }: { flower: ProductWithStock; pastry: ProductWithStock }) {
  const lines = useCart();
  const inCart = (id: number) => lines.find((l) => l.productId === id)?.quantity ?? 0;
  const flowers = inCart(flower.id);
  const pastries = inCart(pastry.id);
  const available = (flower.stock ?? 0) > flowers && (pastry.stock ?? 0) > pastries;

  const addBoth = () => {
    for (const p of [flower, pastry]) {
      cart.add({ productId: p.id, name: p.name, category: p.category, priceIdr: p.priceIdr, imageUrl: p.imageUrl });
    }
  };

  return (
    <section aria-labelledby="gift-title" className="mx-auto w-full max-w-[1180px] px-6 py-12">
      <motion.div
        className="relative flex flex-wrap items-center gap-8 overflow-hidden rounded-[32px] bg-forest p-8 text-white"
        initial={{ opacity: 0, y: 28 }}
        whileInView={{ opacity: 1, y: 0 }}
        viewport={{ once: true, margin: "0px 0px -80px 0px" }}
        transition={{ duration: 0.6, ease: [0.22, 1, 0.36, 1] }}
      >
        <div className="absolute -right-5 -top-6 opacity-90">
          <Illustration kind="sprig" size={140} decorative />
        </div>
        <div className="flex gap-3">
          {[flower, pastry].map((p, i) => (
            <motion.div
              key={p.id}
              initial={{ rotate: 0 }}
              whileInView={{ rotate: i === 0 ? -4 : 4, y: i === 0 ? 0 : 18 }}
              viewport={{ once: true }}
              transition={{ delay: 0.3, type: "spring", stiffness: 120, damping: 14 }}
            >
              <Image src={p.imageUrl || "/images/store/flower-buckets.webp"} alt={p.name} width={150} height={150} className="h-[150px] w-[150px] rounded-[22px] object-cover" />
            </motion.div>
          ))}
        </div>
        <div className="min-w-0 flex-[1_1_320px]">
          <h2 id="gift-title" className="font-serif text-4xl font-bold leading-[1.1]">
            {flower.name} and a {pastry.name.toLowerCase()}
          </h2>
          <p className="mt-2.5 text-mint">Make it a gift: both together, one order, one payment.</p>
        </div>
        <div className="flex flex-col items-start gap-2.5">
          <span className="text-[26px] font-bold tabular-nums">{rupiah(flower.priceIdr + pastry.priceIdr)}</span>
          <motion.button
            type="button"
            whileTap={{ scale: 0.95 }}
            disabled={!available}
            onClick={addBoth}
            className="min-h-[50px] rounded-full bg-white px-6 font-bold text-forest hover:bg-forest-tint disabled:opacity-60"
          >
            {available ? (flowers + pastries > 0 ? "Add another pair" : "Add both to cart") : "No more in stock today"}
          </motion.button>
          <p className="min-h-5 text-sm text-mint" aria-live="polite">
            {flowers + pastries > 0 && (
              <>
                In your cart: {flowers} × {flower.name.toLowerCase()}, {pastries} × {pastry.name.toLowerCase()}.{" "}
                <Link href="/cart" className="font-semibold text-white underline underline-offset-2">
                  View cart
                </Link>
              </>
            )}
          </p>
        </div>
      </motion.div>
    </section>
  );
}
