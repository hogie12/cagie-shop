"use client";

import { AnimatePresence, motion } from "motion/react";
import { cart, MAX_QUANTITY, useCart, type CartLine } from "@/lib/cart";

type Props = {
  item: Omit<CartLine, "quantity">;
  /** Units in stock; null when the item is not stocked at all. */
  stock: number | null;
  /** Width and spacing of the control. */
  className?: string;
};

const swap = {
  initial: { opacity: 0, y: 8 },
  animate: { opacity: 1, y: 0 },
  exit: { opacity: 0, y: -8 },
  transition: { duration: 0.16 },
};

/** "Add to cart" until the item is in the cart, then a stepper that shows how
 * many are in it. Synced with the cart everywhere (header badge, cart page). */
export function QuantityControl({ item, stock, className = "w-[156px]" }: Props) {
  const qty = useCart().find((l) => l.productId === item.productId)?.quantity ?? 0;
  const limit = Math.min(stock ?? 0, MAX_QUANTITY);
  const soldOut = limit <= 0;

  return (
    <div className={`relative h-11 shrink-0 ${className}`}>
      <AnimatePresence mode="popLayout" initial={false}>
        {qty === 0 ? (
          <motion.button
            key="add"
            type="button"
            {...swap}
            whileTap={{ scale: 0.95 }}
            disabled={soldOut}
            onClick={() => cart.add(item)}
            className="absolute inset-0 rounded-full border-[1.5px] border-forest bg-white font-semibold text-forest transition-colors hover:bg-forest-tint disabled:border-field disabled:text-muted disabled:hover:bg-white"
          >
            {soldOut ? "Sold out" : "Add to cart"}
          </motion.button>
        ) : (
          <motion.div
            key="stepper"
            {...swap}
            role="group"
            aria-label={`${item.name} in your cart`}
            className="absolute inset-0 flex items-center justify-between rounded-full bg-forest text-white"
          >
            <button
              type="button"
              aria-label={qty === 1 ? `Remove ${item.name} from cart` : `One less ${item.name}`}
              onClick={() => cart.setQuantity(item.productId, qty - 1)}
              className="flex h-11 w-11 items-center justify-center rounded-full text-xl hover:bg-forest-dark"
            >
              <span aria-hidden="true">−</span>
            </button>
            <motion.span
              key={qty}
              initial={{ scale: 1.25 }}
              animate={{ scale: 1 }}
              transition={{ type: "spring", stiffness: 500, damping: 22 }}
              className="text-sm font-semibold tabular-nums"
            >
              {qty} in cart
            </motion.span>
            <button
              type="button"
              aria-label={`One more ${item.name}`}
              disabled={qty >= limit}
              onClick={() => cart.setQuantity(item.productId, qty + 1)}
              className="flex h-11 w-11 items-center justify-center rounded-full text-xl hover:bg-forest-dark disabled:opacity-40 disabled:hover:bg-transparent"
            >
              <span aria-hidden="true">+</span>
            </button>
          </motion.div>
        )}
      </AnimatePresence>
      <span className="sr-only" aria-live="polite">
        {qty > 0 ? `${qty} ${item.name} in your cart${qty >= limit ? ", that is all we have" : ""}` : ""}
      </span>
    </div>
  );
}
