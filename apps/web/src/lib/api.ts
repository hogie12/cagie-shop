// Browser-side calls to the gateway. NEXT_PUBLIC_ values are baked in at build
// time, so this must be written literally (see Next's environment variables guide).
export const API_URL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

export class ApiError extends Error {
  constructor(
    public status: number,
    message: string,
  ) {
    super(message);
  }
}

type Options = {
  method?: string;
  body?: unknown;
  token?: string | null;
  /** Send cookies: only the /auth routes need this (the refresh cookie). */
  withCookies?: boolean;
};

export async function request<T>(path: string, { method = "GET", body, token, withCookies }: Options = {}): Promise<T> {
  const headers: Record<string, string> = {};
  if (body !== undefined) headers["Content-Type"] = "application/json";
  if (token) headers.Authorization = `Bearer ${token}`;

  let res: Response;
  try {
    res = await fetch(API_URL + path, {
      method,
      headers,
      body: body === undefined ? undefined : JSON.stringify(body),
      credentials: withCookies ? "include" : "omit",
    });
  } catch {
    throw new ApiError(0, "Cannot reach the shop right now. Check your connection and try again.");
  }
  if (res.status === 204) return undefined as T;
  const data = await res.json().catch(() => ({}));
  if (!res.ok) throw new ApiError(res.status, typeof data.error === "string" ? data.error : res.statusText);
  return data as T;
}
