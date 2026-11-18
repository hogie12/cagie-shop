"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { AnimatePresence, motion } from "motion/react";
import { useAuth } from "@/lib/auth";
import { cartTotals, useCart } from "@/lib/cart";
import { Logo } from "./Logo";

const navLink = "inline-flex min-h-11 items-center font-medium text-muted hover:text-ink";

export function SiteHeader() {
  const { status, user, signOut } = useAuth();
  const { count } = cartTotals(useCart());
  // The shadow appears only once content scrolls under the header: it marks
  // that the header now sits above the page.
  const [scrolled, setScrolled] = useState(false);

  useEffect(() => {
    const onScroll = () => setScrolled(window.scrollY > 4);
    onScroll();
    window.addEventListener("scroll", onScroll, { passive: true });
    return () => window.removeEventListener("scroll", onScroll);
  }, []);

  return (
    <header
      className={`sticky top-0 z-40 border-b bg-ground transition-shadow duration-200 ${
        scrolled ? "border-line shadow-[0_8px_24px_-16px_rgba(43,38,34,0.45)]" : "border-line"
      }`}
    >
      <div className="mx-auto flex max-w-[1180px] flex-wrap items-center justify-between gap-x-3 px-4 pt-2 sm:px-6 md:flex-nowrap md:gap-x-8 md:py-2.5">
        <Link href="/" aria-label="Cagie, home" className="text-ink">
          <Logo />
        </Link>

        {/* On phones the links form one row under the logo that scrolls sideways, so the sticky header stays short. */}
        <nav aria-label="Main" className="order-last -mx-4 w-[calc(100%+2rem)] overflow-x-auto px-4 sm:-mx-6 sm:w-[calc(100%+3rem)] sm:px-6 md:order-none md:mx-0 md:w-auto md:grow md:overflow-visible md:px-0">
          <ul className="flex gap-6 whitespace-nowrap">
            <li>
              <Link href="/#products" className={navLink}>Shop</Link>
            </li>
            <li>
              <Link href="/#story" className={navLink}>Our story</Link>
            </li>
            <li>
              <Link href="/#visit" className={navLink}>Visit</Link>
            </li>
            {status === "signedIn" && (
              <li>
                <Link href="/orders" className={navLink}>My orders</Link>
              </li>
            )}
            {user?.role === "admin" && (
              <li>
                <Link href="/admin" className={navLink}>Admin</Link>
              </li>
            )}
          </ul>
        </nav>

        <div className="flex items-center gap-2">
          {status === "signedIn" ? (
            <button type="button" onClick={() => void signOut()} className="min-h-11 px-2 font-medium text-ink hover:text-forest sm:px-3">
              Sign out
            </button>
          ) : (
            <Link href="/login" className="inline-flex min-h-11 items-center px-2 font-medium text-ink hover:text-forest sm:px-3">
              Sign in
            </Link>
          )}
          <Link href="/cart" className="inline-flex min-h-11 items-center gap-2 rounded-full bg-forest px-3.5 font-semibold text-white hover:bg-forest-dark sm:px-4">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
              <path d="M6 7h12l-1 13H7L6 7z" />
              <path d="M9 7a3 3 0 0 1 6 0" />
            </svg>
            <span className="sr-only sm:not-sr-only">Cart</span>
            <span className="relative inline-flex h-[22px] min-w-[22px] items-center justify-center overflow-hidden rounded-full bg-white px-1 text-xs font-bold text-forest tabular-nums">
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
