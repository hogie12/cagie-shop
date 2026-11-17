"use client";

import Image from "next/image";
import { useState } from "react";
import { motion } from "motion/react";
import { cart } from "@/lib/cart";
import { rupiah } from "@/lib/format";
import type { ProductWithStock } from "@/lib/types";
import { Illustration } from "../Illustration";

export function GiftPairing({ flower, pastry }: { flower: ProductWithStock; pastry: ProductWithStock }) {
  const [added, setAdded] = useState(false);
  const available = (flower.stock ?? 0) > 0 && (pastry.stock ?? 0) > 0;

  const addBoth = () => {
    for (const p of [flower, pastry]) {
      cart.add({ productId: p.id, name: p.name, category: p.category, priceIdr: p.priceIdr, imageUrl: p.imageUrl });
    }
    setAdded(true);
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
          <Illustration kind="sprig" size={140} decorative sway />
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
          <p className="font-hand text-[28px] text-blush">make it a gift</p>
          <h2 id="gift-title" className="font-serif text-4xl font-bold leading-[1.1]">
            {flower.name} and a {pastry.name.toLowerCase()}
          </h2>
          <p className="mt-2.5 text-mint">Both together. One order, one payment.</p>
        </div>
        <div className="flex flex-col items-start gap-2.5">
          <span className="text-[26px] font-bold tabular-nums">{rupiah(flower.priceIdr + pastry.priceIdr)}</span>
          <motion.button
            type="button"
            whileTap={{ scale: 0.95 }}
            disabled={!available}
            onClick={addBoth}
            className="min-h-[50px] cursor-pointer rounded-full bg-white px-6 font-bold text-forest disabled:cursor-not-allowed disabled:opacity-60"
          >
            {!available ? "Not available today" : added ? "Added to your cart" : "Add both to cart"}
          </motion.button>
        </div>
      </motion.div>
    </section>
  );
}
