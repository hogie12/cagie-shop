import type { Role } from "./types";

export type Claims = { sub: string; role: Role; exp: number };

/** Reads the access token's payload for the UI (who is signed in, which role).
 * This does NOT verify the signature: the gateway does that on every request,
 * so a forged token only fools this page, never the API. */
export function decodeClaims(token: string): Claims | null {
  const part = token.split(".")[1];
  if (!part) return null;
  try {
    const json = atob(part.replace(/-/g, "+").replace(/_/g, "/").padEnd(Math.ceil(part.length / 4) * 4, "="));
    const c = JSON.parse(json);
    if (typeof c.sub !== "string" || (c.role !== "customer" && c.role !== "admin") || typeof c.exp !== "number") return null;
    return { sub: c.sub, role: c.role, exp: c.exp };
  } catch {
    return null;
  }
}
