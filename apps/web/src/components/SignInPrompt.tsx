import Link from "next/link";
import { Illustration } from "./Illustration";

export function SignInPrompt({ next, text }: { next: string; text: string }) {
  return (
    <div className="flex flex-col items-center gap-3 rounded-3xl border border-line bg-white p-12 text-center">
      <Illustration kind="lily" size={88} decorative />
      <p className="text-muted">{text}</p>
      <Link href={`/login?next=${encodeURIComponent(next)}`} className="inline-flex min-h-11 items-center rounded-full bg-forest px-5 font-semibold text-white">
        Sign in
      </Link>
    </div>
  );
}
