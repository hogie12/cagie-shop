"use client";

import { useRouter, useSearchParams } from "next/navigation";
import { useState } from "react";
import { AnimatePresence, LayoutGroup, motion } from "motion/react";
import { ApiError } from "@/lib/api";
import { useAuth } from "@/lib/auth";

type Mode = "signIn" | "register";

/** Only same-site paths, so ?next= cannot send people to another website. */
export function safeNext(next: string | null): string {
  return next && next.startsWith("/") && !next.startsWith("//") && !next.startsWith("/\\") ? next : "/";
}

function message(err: unknown, mode: Mode): string {
  if (!(err instanceof ApiError)) return "Something went wrong. Please try again.";
  if (err.status === 0) return err.message;
  if (err.status === 401) return "Invalid email or password. Check both and try again.";
  if (err.status === 409) return "That email already has an account. Sign in instead.";
  if (err.status === 429) return "Too many attempts. Wait a minute and try again.";
  if (err.status === 400) return mode === "register" ? "Enter a valid email and a password of at least 8 characters." : "Enter a valid email and password.";
  return "Something went wrong. Please try again.";
}

export function LoginForm() {
  const { signIn, register } = useAuth();
  const router = useRouter();
  const next = safeNext(useSearchParams().get("next"));
  const [mode, setMode] = useState<Mode>("signIn");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setBusy(true);
    setError(null);
    try {
      await (mode === "signIn" ? signIn(email, password) : register(email, password));
      router.push(next);
    } catch (err) {
      setError(message(err, mode));
      setBusy(false);
    }
  };

  return (
    <form onSubmit={submit} className="flex w-full max-w-[380px] flex-col gap-5" noValidate>
      <div>
        <p className="font-hand text-[26px] text-rose">{mode === "signIn" ? "welcome back" : "nice to meet you"}</p>
        <h2 className="font-serif text-[34px] font-bold">{mode === "signIn" ? "Sign in" : "Create account"}</h2>
      </div>

      <LayoutGroup id="auth-mode">
        <div role="group" aria-label="Choose" className="grid grid-cols-2 rounded-full bg-sand p-1">
          {(["signIn", "register"] as const).map((m) => (
            <button
              key={m}
              type="button"
              aria-pressed={mode === m}
              onClick={() => {
                setMode(m);
                setError(null);
              }}
              className={`relative min-h-10 cursor-pointer rounded-full text-sm font-semibold ${mode === m ? "text-white" : "text-ink"}`}
            >
              {mode === m && <motion.span layoutId="auth-pill" className="absolute inset-0 rounded-full bg-ink" transition={{ type: "spring", stiffness: 420, damping: 34 }} />}
              <span className="relative">{m === "signIn" ? "Sign in" : "Create account"}</span>
            </button>
          ))}
        </div>
      </LayoutGroup>

      <div className="flex flex-col gap-1.5">
        <label htmlFor="email" className="text-sm font-semibold">
          Email
        </label>
        <input
          id="email"
          type="email"
          autoComplete="email"
          required
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          placeholder="you@example.com"
          className="min-h-[46px] rounded-xl border border-field bg-white px-3.5"
        />
      </div>
      <div className="flex flex-col gap-1.5">
        <label htmlFor="password" className="text-sm font-semibold">
          Password
        </label>
        <input
          id="password"
          type="password"
          autoComplete={mode === "signIn" ? "current-password" : "new-password"}
          required
          minLength={8}
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          aria-describedby="password-hint"
          className="min-h-[46px] rounded-xl border border-field bg-white px-3.5"
        />
        <span id="password-hint" className="text-[13px] text-muted">
          At least 8 characters.
        </span>
      </div>

      <AnimatePresence>
        {error && (
          <motion.p
            role="alert"
            initial={{ opacity: 0, x: -8 }}
            animate={{ opacity: 1, x: [0, -6, 6, -3, 0] }}
            exit={{ opacity: 0 }}
            transition={{ duration: 0.35 }}
            className="rounded-xl bg-rose-tint px-3.5 py-3 text-sm text-rose-deep"
          >
            {error}
          </motion.p>
        )}
      </AnimatePresence>

      <motion.button
        type="submit"
        whileTap={{ scale: 0.97 }}
        disabled={busy}
        className="min-h-[50px] cursor-pointer rounded-full bg-forest font-semibold text-white hover:bg-forest-dark disabled:opacity-70"
      >
        {busy ? "One moment…" : mode === "signIn" ? "Sign in" : "Create account"}
      </motion.button>
      <p className="text-center text-[13px] text-muted">Too many attempts in a minute are paused for a short while.</p>
    </form>
  );
}
