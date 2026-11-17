"use client";

import { Illustration } from "@/components/Illustration";

// In this Next version an error page receives `retry` (it used to be `reset`).
export default function ErrorPage({ retry }: { error: unknown; retry: () => void }) {
  return (
    <main className="mx-auto flex w-full max-w-[640px] flex-col items-center gap-3 px-6 py-20 text-center">
      <Illustration kind="sprig" size={96} decorative />
      <h1 className="font-serif text-4xl font-bold">Something went wrong</h1>
      <p className="text-muted">This page could not be shown. Try again, or come back in a minute.</p>
      <button type="button" onClick={retry} className="mt-2 min-h-11 cursor-pointer rounded-full bg-forest px-5 font-semibold text-white">
        Try again
      </button>
    </main>
  );
}
