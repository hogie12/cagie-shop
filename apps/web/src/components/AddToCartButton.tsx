"use client";

import { useEffect, useState } from "react";
import { AnimatePresence, motion } from "motion/react";
import { cart, type CartLine } from "@/lib/cart";

type Props = {
  item: Omit<CartLine, "quantity">;
  quantity?: number;
  disabled?: boolean;
  label?: string;
  className?: string;
};

export function AddToCartButton({ item, quantity = 1, disabled, label = "Add to cart", className }: Props) {
  const [added, setAdded] = useState(false);

  useEffect(() => {
    if (!added) return;
    const t = setTimeout(() => setAdded(false), 1400);
    return () => clearTimeout(t);
  }, [added]);

  return (
    <motion.button
      type="button"
      disabled={disabled}
      whileTap={{ scale: 0.94 }}
      onClick={() => {
        cart.add(item, quantity);
        setAdded(true);
      }}
      className={
        className ??
        "relative min-h-11 min-w-[124px] cursor-pointer overflow-hidden rounded-full border-[1.5px] border-forest bg-white px-4 font-semibold text-forest transition-colors hover:bg-forest-tint disabled:cursor-not-allowed disabled:opacity-50"
      }
    >
      <AnimatePresence mode="wait" initial={false}>
        <motion.span
          key={added ? "added" : "idle"}
          className="inline-flex items-center justify-center gap-1.5"
          initial={{ y: 14, opacity: 0 }}
          animate={{ y: 0, opacity: 1 }}
          exit={{ y: -14, opacity: 0 }}
          transition={{ duration: 0.18 }}
        >
          {added ? (
            <>
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.6" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                <path d="M5 12l5 5 9-10" />
              </svg>
              Added
            </>
          ) : disabled ? (
            "Sold out"
          ) : (
            label
          )}
        </motion.span>
      </AnimatePresence>
      <span className="sr-only" aria-live="polite">
        {added ? `${item.name} added to cart` : ""}
      </span>
    </motion.button>
  );
}
