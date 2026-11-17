"use client";

import { AnimatePresence, motion } from "motion/react";
import type { OrderView } from "@/lib/orderStatus";

/** The four saga steps. The line between steps fills as the order moves on,
 * the current step pulses, and finished steps pop in with a check mark. */
export function OrderTracker({ steps }: { steps: OrderView["steps"] }) {
  return (
    <ol aria-label="Order progress" className="m-0 grid list-none grid-cols-[repeat(auto-fit,minmax(150px,1fr))] gap-4 p-0">
      {steps.map((s, i) => (
        <li key={s.label} className="flex flex-col gap-2.5" aria-current={s.state === "current" ? "step" : undefined}>
          <div className="flex items-center gap-2.5">
            <motion.span
              className="relative inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-full"
              animate={{
                backgroundColor: s.state === "done" ? "#1F4D3A" : s.state === "failed" ? "#B23A5B" : "#FFFFFF",
                borderColor: s.state === "todo" ? "#D8D0C3" : s.state === "current" ? "#1F4D3A" : "rgba(0,0,0,0)",
              }}
              style={{ borderWidth: 2, borderStyle: "solid" }}
              transition={{ duration: 0.3 }}
            >
              <AnimatePresence mode="wait">
                {s.state === "done" && (
                  <motion.svg key="done" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="#FFFFFF" strokeWidth="2.6" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                    <motion.path d="M5 12l5 5 9-10" initial={{ pathLength: 0 }} animate={{ pathLength: 1 }} transition={{ duration: 0.35, delay: 0.1 }} />
                  </motion.svg>
                )}
                {s.state === "failed" && (
                  <motion.svg key="failed" initial={{ scale: 0 }} animate={{ scale: 1 }} width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="#FFFFFF" strokeWidth="2.6" strokeLinecap="round" aria-hidden="true">
                    <path d="M6 6l12 12" />
                    <path d="M18 6L6 18" />
                  </motion.svg>
                )}
                {s.state === "current" && (
                  <motion.span
                    key="current"
                    aria-hidden="true"
                    className="h-2.5 w-2.5 rounded-full bg-forest"
                    animate={{ scale: [1, 1.5, 1], opacity: [1, 0.55, 1] }}
                    transition={{ duration: 1.2, repeat: Infinity, ease: "easeInOut" }}
                  />
                )}
              </AnimatePresence>
            </motion.span>
            {i < steps.length - 1 && (
              <span className="relative h-0.5 grow overflow-hidden rounded bg-line">
                <motion.span
                  className="absolute inset-y-0 left-0 rounded bg-forest"
                  initial={false}
                  animate={{ width: s.state === "done" ? "100%" : "0%" }}
                  transition={{ duration: 0.6, ease: "easeInOut" }}
                />
              </span>
            )}
          </div>
          <div>
            <p className="font-semibold">{s.label}</p>
            <p className={`text-sm ${s.state === "failed" ? "text-rose-dark" : "text-muted"}`}>{s.note}</p>
          </div>
        </li>
      ))}
    </ol>
  );
}
