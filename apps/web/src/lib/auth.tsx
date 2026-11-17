"use client";

import { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
import { ApiError, request } from "./api";
import { decodeClaims } from "./jwt";
import type { Role } from "./types";

type User = { id: string; role: Role };
type Status = "loading" | "signedOut" | "signedIn";
type TokenResponse = { accessToken: string; expiresIn: number };

type AuthContextValue = {
  status: Status;
  user: User | null;
  signIn: (email: string, password: string) => Promise<void>;
  register: (email: string, password: string) => Promise<void>;
  signOut: () => Promise<void>;
  /** Calls the API with the access token, renewing it once if it expired. */
  authRequest: <T>(path: string, init?: { method?: string; body?: unknown }) => Promise<T>;
};

const AuthContext = createContext<AuthContextValue | null>(null);

// One refresh at a time for the whole page. Refresh tokens are single-use: two
// parallel refreshes (React runs effects twice in development) would present
// the same token twice, which the auth service treats as theft and revokes
// every session.
let refreshInFlight: Promise<TokenResponse | null> | null = null;

function refreshOnce(): Promise<TokenResponse | null> {
  refreshInFlight ??= request<TokenResponse>("/auth/refresh", { method: "POST", withCookies: true })
    .catch((err) => {
      if (err instanceof ApiError && err.status === 401) return null;
      throw err;
    })
    .finally(() => {
      refreshInFlight = null;
    });
  return refreshInFlight;
}

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [status, setStatus] = useState<Status>("loading");
  const [user, setUser] = useState<User | null>(null);
  // The access token lives only in memory, never in localStorage.
  const tokenRef = useRef<string | null>(null);
  // When to renew the access token; a new object each time restarts the timer.
  const [renewal, setRenewal] = useState<{ inMs: number } | null>(null);

  const accept = useCallback((res: TokenResponse | null) => {
    const claims = res ? decodeClaims(res.accessToken) : null;
    if (!res || !claims) {
      tokenRef.current = null;
      setUser(null);
      setStatus("signedOut");
      setRenewal(null);
      return;
    }
    tokenRef.current = res.accessToken;
    setUser({ id: claims.sub, role: claims.role });
    setStatus("signedIn");
    // Renew a minute before the 15-minute token expires.
    setRenewal({ inMs: Math.max(res.expiresIn - 60, 30) * 1000 });
  }, []);

  useEffect(() => {
    refreshOnce().then(accept, () => accept(null));
  }, [accept]);

  useEffect(() => {
    if (!renewal) return;
    const t = setTimeout(() => void refreshOnce().then(accept, () => undefined), renewal.inMs);
    return () => clearTimeout(t);
  }, [renewal, accept]);

  const signIn = useCallback(
    async (email: string, password: string) => {
      accept(await request<TokenResponse>("/auth/login", { method: "POST", body: { email, password }, withCookies: true }));
    },
    [accept],
  );

  const register = useCallback(
    async (email: string, password: string) => {
      await request("/auth/register", { method: "POST", body: { email, password } });
      await signIn(email, password);
    },
    [signIn],
  );

  const signOut = useCallback(async () => {
    await request("/auth/logout", { method: "POST", withCookies: true }).catch(() => undefined);
    accept(null);
  }, [accept]);

  const authRequest = useCallback(
    async <T,>(path: string, init: { method?: string; body?: unknown } = {}): Promise<T> => {
      try {
        return await request<T>(path, { ...init, token: tokenRef.current });
      } catch (err) {
        if (!(err instanceof ApiError) || err.status !== 401) throw err;
        const renewed = await refreshOnce();
        accept(renewed);
        if (!renewed) throw err;
        return request<T>(path, { ...init, token: renewed.accessToken });
      }
    },
    [accept],
  );

  const value = useMemo(() => ({ status, user, signIn, register, signOut, authRequest }), [status, user, signIn, register, signOut, authRequest]);
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used inside AuthProvider");
  return ctx;
}
