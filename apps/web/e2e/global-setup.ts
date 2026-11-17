import { mkdirSync, writeFileSync } from "node:fs";
import { dirname } from "node:path";
import { randomBytes } from "node:crypto";
import { request } from "@playwright/test";
import { adminCredentials, GATEWAY, STATE_FILE, type Accounts } from "./support";

// Signs in the admin once and creates one fresh customer for the whole run.
export default async function globalSetup() {
  const api = await request.newContext();
  const admin = adminCredentials();
  const login = await api.post(`${GATEWAY}/auth/login`, { data: admin });
  if (!login.ok()) throw new Error(`admin login failed: ${login.status()} (is the backend running?)`);
  const adminToken = (await login.json()).accessToken as string;

  const customerEmail = `e2e-${Date.now()}@example.com`;
  const customerPassword = randomBytes(12).toString("hex");
  const reg = await api.post(`${GATEWAY}/auth/register`, { data: { email: customerEmail, password: customerPassword } });
  if (reg.status() !== 201) throw new Error(`customer registration failed: ${reg.status()}`);

  const state: Accounts = { adminToken, customerEmail, customerPassword };
  mkdirSync(dirname(STATE_FILE), { recursive: true });
  writeFileSync(STATE_FILE, JSON.stringify(state));
  await api.dispose();
}
