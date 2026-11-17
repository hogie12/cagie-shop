"use client";

import Image from "next/image";
import Link from "next/link";
import { motion, type Variants } from "motion/react";
import { Illustration } from "../Illustration";
import type { ProductWithStock } from "@/lib/types";

const container: Variants = { hidden: {}, show: { transition: { staggerChildren: 0.09 } } };
const item: Variants = {
  hidden: { opacity: 0, y: 24 },
  show: { opacity: 1, y: 0, transition: { duration: 0.6, ease: [0.22, 1, 0.36, 1] } },
};

/** Spotlight: the product with the least stock (if any is running low). */
export function Hero({ spotlight }: { spotlight: ProductWithStock | null }) {
  return (
    <section className="mx-auto flex max-w-[1180px] flex-wrap items-center gap-10 px-6 pb-14 pt-12">
      <motion.div className="min-w-0 flex-[1_1_460px]" variants={container} initial="hidden" animate="show">
        <motion.p variants={item} className="mb-1 font-hand text-[30px] text-rose">
          baked daily, arranged to order
        </motion.p>
        <motion.h1 variants={item} className="font-serif text-[clamp(40px,6vw,62px)] font-bold leading-[1.02] tracking-[-0.02em]">
          Pastries in the morning, <em className="font-semibold text-forest">flowers</em> all day.
        </motion.h1>
        <motion.p variants={item} className="mt-5 max-w-[500px] text-lg text-muted">
          A small shop for two of life&apos;s easy pleasures. Order a box of pastries, a bouquet, or both, and follow your order live while Cagie gets it ready.
        </motion.p>
        <motion.div variants={item} className="mt-7 flex flex-wrap gap-3">
          <Link href="#products" className="inline-flex min-h-[50px] items-center rounded-full bg-forest px-6 font-semibold text-white hover:bg-forest-dark">
            Shop pastries
          </Link>
          <Link href="#products" className="inline-flex min-h-[50px] items-center rounded-full border-[1.5px] border-forest px-6 font-semibold text-forest hover:bg-forest-tint">
            Shop flowers
          </Link>
        </motion.div>
      </motion.div>

      <motion.div
        className="relative min-w-0 flex-[1_1_460px]"
        initial={{ opacity: 0, scale: 0.96 }}
        animate={{ opacity: 1, scale: 1 }}
        transition={{ duration: 0.8, ease: [0.22, 1, 0.36, 1] }}
      >
        <Image
          src="/images/store/hero-pastry-display.webp"
          alt="A bakery display case with fruit tarts and cakes"
          width={1800}
          height={1200}
          priority
          sizes="(max-width: 1000px) 100vw, 560px"
          className="aspect-[5/4] w-full rounded-[32px] object-cover"
        />
        <div className="absolute -bottom-8 -left-9">
          <Illustration kind="sprig" size={130} decorative sway />
        </div>
        {spotlight && (
          <motion.div
            className="absolute bottom-5 right-5 flex items-center gap-3 rounded-[18px] bg-white py-2.5 pl-2.5 pr-4 shadow-[0_10px_30px_rgba(43,38,34,0.14)]"
            initial={{ opacity: 0, y: 16 }}
            animate={{ opacity: 1, y: [0, -6, 0] }}
            transition={{ opacity: { delay: 0.6, duration: 0.4 }, y: { delay: 1, duration: 4, repeat: Infinity, ease: "easeInOut" } }}
          >
            {spotlight.imageUrl && <Image src={spotlight.imageUrl} alt="" width={52} height={52} className="h-[52px] w-[52px] rounded-xl object-cover" />}
            <span className="flex flex-col leading-tight">
              <span className="text-xs text-muted">Only {spotlight.stock} left</span>
              <span className="font-bold">{spotlight.name}</span>
            </span>
          </motion.div>
        )}
      </motion.div>
    </section>
  );
}
