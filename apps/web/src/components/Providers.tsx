"use client";

import { MotionConfig } from "motion/react";
import { AuthProvider } from "@/lib/auth";

export function Providers({ children }: { children: React.ReactNode }) {
  // reducedMotion="user": every motion animation follows the visitor's
  // "reduce motion" setting (transforms off, fades kept).
  return (
    <MotionConfig reducedMotion="user">
      <AuthProvider>{children}</AuthProvider>
    </MotionConfig>
  );
}
