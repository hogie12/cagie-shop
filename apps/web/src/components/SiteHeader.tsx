"use client";

import Link from "next/link";
import { AnimatePresence, motion } from "motion/react";
import { useAuth } from "@/lib/auth";
import { cartTotals, useCart } from "@/lib/cart";
import { Illustration } from "./Illustration";

export function SiteHeader() {
  const { status, user, signOut } = useAuth();
  const { count } = cartTotals(useCart());

  return (
    <header className="border-b border-line bg-ground">
      <div className="mx-auto flex max-w-[1180px] flex-wrap items-center gap-x-8 gap-y-3 px-6 py-3.5">
        <Link href="/" className="flex items-center gap-2.5 text-ink no-underline">
          <Illustration kind="rose" size={36} decorative />
          <span className="flex flex-col leading-none">
            <span className="font-serif text-[28px] font-bold text-forest">Cagie</span>
            <span className="font-hand text-lg text-rose">pastry &amp; flowers</span>
          </span>
        </Link>

        <nav aria-label="Main" className="flex grow flex-wrap gap-x-6 gap-y-2">
          <Link href="/#products" className="font-semibold text-ink">Shop</Link>
          <Link href="/#story" className="font-medium text-muted hover:text-ink">Our story</Link>
          <Link href="/#visit" className="font-medium text-muted hover:text-ink">Visit</Link>
          {status === "signedIn" && <Link href="/orders" className="font-medium text-muted hover:text-ink">My orders</Link>}
          {user?.role === "admin" && <Link href="/admin" className="font-medium text-muted hover:text-ink">Admin</Link>}
        </nav>

        <div className="flex items-center gap-3">
          {status === "signedIn" ? (
            <button type="button" onClick={() => void signOut()} className="min-h-11 cursor-pointer px-3 font-medium text-ink">
              Sign out
            </button>
          ) : (
            <Link href="/login" className="px-3 py-2.5 font-medium text-ink">Sign in</Link>
          )}
          <Link href="/cart" className="inline-flex min-h-11 items-center gap-2 rounded-full bg-forest px-4 font-semibold text-white hover:bg-forest-dark">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
              <path d="M6 7h12l-1 13H7L6 7z" />
              <path d="M9 7a3 3 0 0 1 6 0" />
            </svg>
            <span>Cart</span>
            <span className="relative inline-flex h-[22px] min-w-[22px] items-center justify-center overflow-hidden rounded-full bg-white px-1 text-xs font-bold text-forest">
              <span className="sr-only">items:</span>
              {/* A new number pops in each time the count changes. */}
              <AnimatePresence mode="popLayout" initial={false}>
                <motion.span
                  key={count}
                  initial={{ y: 12, scale: 0.6, opacity: 0 }}
                  animate={{ y: 0, scale: 1, opacity: 1 }}
                  exit={{ y: -12, opacity: 0 }}
                  transition={{ type: "spring", stiffness: 500, damping: 24 }}
                >
                  {count}
                </motion.span>
              </AnimatePresence>
            </span>
          </Link>
        </div>
      </div>
    </header>
  );
}
