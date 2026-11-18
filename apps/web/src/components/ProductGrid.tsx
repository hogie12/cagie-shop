"use client";

import Image from "next/image";
import { useState } from "react";
import { AnimatePresence, LayoutGroup, motion } from "motion/react";
import { rupiah, stockLabel } from "@/lib/format";
import { illustrationFor } from "@/lib/illustrations";
import type { Category, ProductWithStock } from "@/lib/types";
import { Illustration } from "./Illustration";
import { QuantityControl } from "./QuantityControl";

type Filter = "all" | Category;

const tint: Record<string, string> = {
  croissant: "bg-butter",
  eclair: "bg-butter",
  tart: "bg-butter",
  rose: "bg-rose-tint",
  sunflower: "bg-sun",
  lily: "bg-forest-tint",
  sprig: "bg-forest-tint",
};

export function ProductGrid({ products }: { products: ProductWithStock[] }) {
  const [filter, setFilter] = useState<Filter>("all");
  const tabs: { id: Filter; label: string; count: number }[] = [
    { id: "all", label: "All", count: products.length },
    { id: "pastry", label: "Pastries", count: products.filter((p) => p.category === "pastry").length },
    { id: "flower", label: "Flowers", count: products.filter((p) => p.category === "flower").length },
  ];
  const shown = products.filter((p) => filter === "all" || p.category === filter);

  return (
    <>
      <div className="mb-6 flex flex-wrap items-end justify-between gap-4">
        <div>
          <h2 id="products-title" className="font-serif text-[40px] font-bold leading-tight">
            Shop the case
          </h2>
        </div>
        <LayoutGroup id="category">
          <div role="group" aria-label="Filter by category" className="inline-flex gap-1 rounded-full bg-sand p-1">
            {tabs.map((t) => {
              const on = t.id === filter;
              return (
                <button
                  key={t.id}
                  type="button"
                  aria-pressed={on}
                  onClick={() => setFilter(t.id)}
                  className={`relative min-h-10 cursor-pointer rounded-full px-4 text-sm font-semibold transition-colors ${on ? "text-white" : "text-ink"}`}
                >
                  {/* The dark pill slides between tabs instead of jumping. */}
                  {on && <motion.span layoutId="active-tab" className="absolute inset-0 rounded-full bg-ink" transition={{ type: "spring", stiffness: 420, damping: 34 }} />}
                  <span className="relative">
                    {t.label} <span className="font-medium opacity-70">{t.count}</span>
                  </span>
                </button>
              );
            })}
          </div>
        </LayoutGroup>
      </div>

      <motion.ul layout className="grid list-none grid-cols-[repeat(auto-fill,minmax(260px,1fr))] gap-6 p-0">
        <AnimatePresence mode="popLayout">
          {shown.map((p, i) => {
            const kind = illustrationFor(p.name, p.category);
            return (
              <motion.li
                key={p.id}
                layout
                initial={{ opacity: 0, y: 24, scale: 0.97 }}
                animate={{ opacity: 1, y: 0, scale: 1 }}
                exit={{ opacity: 0, scale: 0.94, transition: { duration: 0.18 } }}
                transition={{ duration: 0.45, delay: i * 0.05, ease: [0.22, 1, 0.36, 1] }}
                className="group flex flex-col overflow-hidden rounded-3xl border border-line bg-white"
              >
                <div className={`relative aspect-[4/3] overflow-hidden ${tint[kind]}`}>
                  {p.imageUrl ? (
                    <Image
                      src={p.imageUrl}
                      alt={p.name}
                      fill
                      sizes="(max-width: 640px) 100vw, (max-width: 1100px) 50vw, 380px"
                      className="object-cover transition-transform duration-500 ease-out group-hover:scale-105 motion-reduce:transition-none motion-reduce:group-hover:scale-100"
                    />
                  ) : (
                    <div className="flex h-full items-center justify-center">
                      <Illustration kind={kind} size={150} />
                    </div>
                  )}
                  {p.stock !== null && p.stock > 0 && p.stock <= 5 && (
                    <span className="absolute right-3 top-3 -rotate-3 rounded-full bg-white px-3 font-hand text-[21px] text-rose-dark">almost gone</span>
                  )}
                </div>
                <div className="flex grow flex-col gap-1.5 px-5 pb-5 pt-4">
                  <div className="flex items-center justify-between gap-2">
                    <span className={`rounded-full px-2.5 py-0.5 text-xs font-semibold ${p.category === "pastry" ? "bg-butter text-[#6B4E26]" : "bg-rose-tint text-rose-dark"}`}>
                      {p.category === "pastry" ? "Pastry" : "Flowers"}
                    </span>
                    <span className="text-[13px] text-muted">{stockLabel(p.stock)}</span>
                  </div>
                  <h3 className="mt-1 font-serif text-[23px] font-bold">{p.name}</h3>
                  <p className="grow text-sm text-muted">{p.description}</p>
                  <span className="mt-2 whitespace-nowrap text-lg font-bold tabular-nums">{rupiah(p.priceIdr)}</span>
                  <QuantityControl
                    className="mt-1 w-full"
                    stock={p.stock}
                    item={{ productId: p.id, name: p.name, category: p.category, priceIdr: p.priceIdr, imageUrl: p.imageUrl }}
                  />
                </div>
              </motion.li>
            );
          })}
        </AnimatePresence>
      </motion.ul>
    </>
  );
}
