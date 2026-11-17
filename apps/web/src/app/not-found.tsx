import Link from "next/link";
import { Illustration } from "@/components/Illustration";

export default function NotFound() {
  return (
    <main className="mx-auto flex w-full max-w-[640px] flex-col items-center gap-3 px-6 py-20 text-center">
      <Illustration kind="eclair" size={96} decorative />
      <p className="font-hand text-[26px] text-rose">nothing here</p>
      <h1 className="font-serif text-4xl font-bold">Page not found</h1>
      <p className="text-muted">This page does not exist. The pastries and flowers are this way.</p>
      <Link href="/" className="mt-2 inline-flex min-h-11 items-center rounded-full bg-forest px-5 font-semibold text-white">
        Back to the shop
      </Link>
    </main>
  );
}
