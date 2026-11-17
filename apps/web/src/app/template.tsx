"use client";

import { motion } from "motion/react";

// A template remounts on every navigation, so each page fades in gently.
export default function Template({ children }: { children: React.ReactNode }) {
  return (
    <motion.div className="flex grow flex-col" initial={{ opacity: 0, y: 12 }} animate={{ opacity: 1, y: 0 }} transition={{ duration: 0.35, ease: "easeOut" }}>
      {children}
    </motion.div>
  );
}
