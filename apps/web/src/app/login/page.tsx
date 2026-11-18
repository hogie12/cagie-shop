import { Suspense } from "react";
import type { Metadata } from "next";
import { Illustration } from "@/components/Illustration";
import { LoginForm } from "./LoginForm";

export const metadata: Metadata = { title: "Sign in" };

export default function LoginPage() {
  return (
    <main className="flex grow flex-wrap">
      <section className="relative flex min-w-0 flex-[1_1_420px] flex-col gap-8 overflow-hidden bg-forest p-12 text-white">
        <div>
          <h1 className="max-w-[440px] font-serif text-[clamp(34px,4vw,46px)] font-bold leading-[1.05]">
            Sign in to order and <em className="font-semibold">follow</em> your orders.
          </h1>
          <p className="mt-4 max-w-[420px] text-mint">Browsing is open to everyone. You need an account to check out, so your order history stays yours.</p>
        </div>
        <div className="relative mt-auto h-[220px] rounded-3xl bg-butter" aria-hidden="true">
          <div className="absolute -bottom-1.5 left-3">
            <Illustration kind="lily" size={150} decorative />
          </div>
          <div className="absolute left-[150px] top-3.5">
            <Illustration kind="sunflower" size={140} decorative />
          </div>
          <div className="absolute bottom-2.5 right-6">
            <Illustration kind="tart" size={140} decorative />
          </div>
        </div>
      </section>
      <section className="flex min-w-0 flex-[1_1_420px] items-center justify-center px-6 py-12">
        {/* useSearchParams (the ?next= link) needs a Suspense boundary in this Next version. */}
        <Suspense fallback={<p className="text-muted">Loading…</p>}>
          <LoginForm />
        </Suspense>
      </section>
    </main>
  );
}
