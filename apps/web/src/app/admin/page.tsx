"use client";

import { useCallback, useEffect, useState } from "react";
import { LayoutGroup, motion } from "motion/react";
import { MoneyInput } from "@/components/MoneyInput";
import { ProductThumb } from "@/components/ProductThumb";
import { SignInPrompt } from "@/components/SignInPrompt";
import { ApiError, request } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { digitsOnly } from "@/lib/format";
import type { Category, Product, ProductWithStock, ProviderMode } from "@/lib/types";

const modes: { id: ProviderMode; label: string; note: string; active: string }[] = [
  { id: "succeed", label: "Approve", note: "Every payment is approved.", active: "bg-forest" },
  { id: "decline", label: "Decline", note: "Every new payment is declined. Those orders are cancelled and their stock goes back.", active: "bg-rose" },
  { id: "down", label: "Outage", note: "The provider does not answer. Payments are retried, then fail as unavailable after about 10 seconds.", active: "bg-ink" },
];

type Draft = { priceIdr: string; stock: string };

/** Every product with its stock (null when the inventory has no record yet). */
async function fetchProducts(): Promise<ProductWithStock[]> {
  const products = await request<Product[]>("/products");
  return Promise.all(
    products.map(async (p) => {
      const stock = await request<{ quantity: number }>(`/inventory/${p.id}`).then(
        (s) => s.quantity,
        () => null,
      );
      return { ...p, stock };
    }),
  );
}

export default function AdminPage() {
  const { status, user, authRequest } = useAuth();
  const isAdmin = status === "signedIn" && user?.role === "admin";
  const [mode, setMode] = useState<ProviderMode | null>(null);
  const [modeError, setModeError] = useState<string | null>(null);
  const [rows, setRows] = useState<ProductWithStock[] | null>(null);
  const [drafts, setDrafts] = useState<Record<number, Draft>>({});
  const [confirmRemove, setConfirmRemove] = useState<number | null>(null);
  const [notice, setNotice] = useState<string | null>(null);
  const [saving, setSaving] = useState<number | null>(null);
  const [saved, setSaved] = useState<number | null>(null);

  useEffect(() => {
    if (saved === null) return;
    const t = setTimeout(() => setSaved(null), 2500);
    return () => clearTimeout(t);
  }, [saved]);

  const show = useCallback((withStock: ProductWithStock[]) => {
    setRows(withStock);
    setDrafts(Object.fromEntries(withStock.map((p) => [p.id, { priceIdr: String(p.priceIdr), stock: p.stock === null ? "" : String(p.stock) }])));
  }, []);
  const load = useCallback(() => fetchProducts().then(show), [show]);

  useEffect(() => {
    if (!isAdmin) return;
    fetchProducts().then(show, () => setNotice("Products could not be loaded. Refresh the page."));
    authRequest<{ mode: ProviderMode }>("/payments/provider").then(
      (r) => setMode(r.mode),
      (err) => setModeError(err instanceof ApiError && err.status === 404 ? "Provider controls are turned off on this server." : "Could not load the provider mode."),
    );
  }, [isAdmin, show, authRequest]);

  const switchMode = async (m: ProviderMode) => {
    const previous = mode;
    setMode(m); // optimistic, the pill slides at once
    try {
      await authRequest("/payments/provider", { method: "PUT", body: { mode: m } });
    } catch {
      setMode(previous);
      setModeError("The mode could not be changed. Try again.");
    }
  };

  const save = async (p: ProductWithStock) => {
    const d = drafts[p.id];
    if (d.priceIdr === "" || d.stock === "") {
      setNotice(`Fill in both price and stock for ${p.name}.`);
      return;
    }
    const price = Number(d.priceIdr);
    const stock = Number(d.stock);
    setSaving(p.id);
    setNotice(null);
    try {
      if (price !== p.priceIdr) {
        await authRequest(`/products/${p.id}`, {
          method: "PUT",
          body: { name: p.name, category: p.category, description: p.description, priceIdr: price, imageUrl: p.imageUrl },
        });
      }
      if (stock !== p.stock) await authRequest(`/inventory/${p.id}`, { method: "PUT", body: { quantity: stock } });
      setSaved(p.id);
      setNotice(`${p.name} saved. The shop page shows it within about a minute.`);
      await load();
    } catch {
      setNotice(`${p.name} could not be saved. Try again.`);
    } finally {
      setSaving(null);
    }
  };

  const remove = async (p: ProductWithStock) => {
    try {
      await authRequest(`/products/${p.id}`, { method: "DELETE" });
      setConfirmRemove(null);
      setNotice(`${p.name} removed from the shop. Past orders keep it.`);
      await load();
    } catch {
      setNotice(`${p.name} could not be removed. Try again.`);
    }
  };

  if (status === "loading") return <main className="mx-auto w-full max-w-[1180px] px-6 pt-8 text-muted">Loading…</main>;
  if (status === "signedOut")
    return (
      <main className="mx-auto w-full max-w-[880px] px-6 pb-16 pt-8">
        <SignInPrompt next="/admin" text="Sign in with an admin account to manage the shop." />
      </main>
    );
  if (!isAdmin)
    return (
      <main className="mx-auto w-full max-w-[880px] px-6 pb-16 pt-8">
        <h1 className="font-serif text-4xl font-bold">Admins only</h1>
        <p className="mt-2 text-muted">This account cannot manage the shop.</p>
      </main>
    );

  return (
    <main className="mx-auto flex w-full max-w-[1180px] flex-col gap-6 px-6 pb-16 pt-8">
      <div>
        <h1 className="font-serif text-[44px] font-bold tracking-[-0.01em]">Shop admin</h1>
      </div>

      <section aria-labelledby="provider-title" className="flex flex-wrap items-center justify-between gap-6 rounded-[22px] border border-line bg-white p-6">
        <div className="min-w-0 flex-[1_1_360px]">
          <h2 id="provider-title" className="font-serif text-[22px] font-bold">
            Payment provider (test mode)
          </h2>
          <p className="mt-1 text-sm text-muted">{modeError ?? modes.find((m) => m.id === mode)?.note ?? "Loading…"}</p>
        </div>
        {mode && (
          <LayoutGroup id="provider">
            <div role="group" aria-label="Provider mode" className="inline-flex gap-1 rounded-full bg-sand p-1">
              {modes.map((m) => (
                <button
                  key={m.id}
                  type="button"
                  aria-pressed={mode === m.id}
                  onClick={() => void switchMode(m.id)}
                  className={`relative min-h-10 cursor-pointer rounded-full px-4 text-sm font-semibold ${mode === m.id ? "text-white" : "text-ink"}`}
                >
                  {mode === m.id && <motion.span layoutId="provider-pill" className={`absolute inset-0 rounded-full ${m.active}`} transition={{ type: "spring", stiffness: 420, damping: 34 }} />}
                  <span className="relative">{m.label}</span>
                </button>
              ))}
            </div>
          </LayoutGroup>
        )}
      </section>

      <section aria-labelledby="products-title" className="rounded-[22px] border border-line bg-white">
        <div className="flex flex-wrap items-center justify-between gap-3 border-b border-line px-6 py-5">
          <div>
            <h2 id="products-title" className="font-serif text-[22px] font-bold">
              Products and stock
            </h2>
            <p className="mt-0.5 text-sm text-muted">Removing a product hides it from the shop; past orders keep it.</p>
          </div>
        </div>
        {notice && (
          <p role="status" className="mx-6 mt-4 rounded-xl bg-forest-tint px-3.5 py-2.5 text-sm text-forest">
            {notice}
          </p>
        )}
        <div className="overflow-x-auto">
          <table className="w-full min-w-[860px] border-collapse text-sm">
            <thead>
              <tr className="text-left text-xs uppercase tracking-wider text-muted">
                <th scope="col" className="px-6 py-3 font-semibold">Product</th>
                <th scope="col" className="px-3 py-3 font-semibold">Category</th>
                <th scope="col" className="px-3 py-3 font-semibold">Price</th>
                <th scope="col" className="px-3 py-3 font-semibold">Stock</th>
                <th scope="col" className="px-6 py-3 text-right font-semibold">Actions</th>
              </tr>
            </thead>
            <tbody>
              {rows === null && (
                <tr>
                  <td colSpan={5} className="px-6 py-6 text-muted">Loading products…</td>
                </tr>
              )}
              {rows?.map((p) => {
                const d = drafts[p.id] ?? { priceIdr: "", stock: "" };
                const low = p.stock !== null && p.stock <= 5;
                const dirty = d.priceIdr !== String(p.priceIdr) || d.stock !== (p.stock === null ? "" : String(p.stock));
                return (
                  <tr key={p.id} className="border-t border-line">
                    <td className="px-6 py-2.5">
                      <span className="flex items-center gap-3 font-semibold">
                        <ProductThumb name={p.name} category={p.category} imageUrl={p.imageUrl} size={44} />
                        {p.name}
                      </span>
                    </td>
                    <td className="px-3 py-2.5">
                      <span className={`rounded-full px-2.5 py-0.5 text-xs font-semibold ${p.category === "pastry" ? "bg-butter text-[#6B4E26]" : "bg-rose-tint text-rose-dark"}`}>
                        {p.category === "pastry" ? "Pastry" : "Flowers"}
                      </span>
                    </td>
                    <td className="px-3 py-2.5">
                      <MoneyInput
                        aria-label={`Price of ${p.name} in rupiah`}
                        value={d.priceIdr}
                        onChange={(v) => setDrafts({ ...drafts, [p.id]: { ...d, priceIdr: v } })}
                      />
                    </td>
                    <td className="px-3 py-2.5">
                      <span className="flex items-center gap-2.5">
                        <input
                          inputMode="numeric"
                          aria-label={`Stock of ${p.name}`}
                          value={d.stock}
                          placeholder="0"
                          maxLength={6}
                          onChange={(e) => setDrafts({ ...drafts, [p.id]: { ...d, stock: digitsOnly(e.target.value) } })}
                          className="min-h-10 w-20 rounded-[10px] border border-field px-2.5 tabular-nums"
                        />
                        {low && <span className="font-hand text-xl text-rose-dark">running low</span>}
                      </span>
                    </td>
                    <td className="whitespace-nowrap px-6 py-2.5 text-right">
                      {confirmRemove === p.id ? (
                        <>
                          <span className="mr-2 text-muted">Remove {p.name}?</span>
                          <button type="button" onClick={() => void remove(p)} className="min-h-10 cursor-pointer px-3 font-semibold text-rose-dark">Yes, remove</button>
                          <button type="button" onClick={() => setConfirmRemove(null)} className="min-h-10 cursor-pointer px-3 font-semibold">Keep</button>
                        </>
                      ) : (
                        <>
                          <button
                            type="button"
                            onClick={() => void save(p)}
                            disabled={!dirty || saving === p.id}
                            className="min-h-10 min-w-[84px] px-3 font-semibold text-forest disabled:text-muted"
                          >
                            {saving === p.id ? "Saving…" : saved === p.id && !dirty ? "Saved" : "Save"}
                          </button>
                          <button type="button" onClick={() => setConfirmRemove(p.id)} className="min-h-10 cursor-pointer px-3 font-semibold text-rose-dark">Remove</button>
                        </>
                      )}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
        <AddProduct
          onAdded={async (name) => {
            setNotice(`${name} added. The shop page shows it within about a minute.`);
            await load();
          }}
          authRequest={authRequest}
        />
      </section>
    </main>
  );
}

function AddProduct({ onAdded, authRequest }: { onAdded: (name: string) => Promise<void>; authRequest: ReturnType<typeof useAuth>["authRequest"] }) {
  const [form, setForm] = useState({ name: "", category: "pastry" as Category, description: "", priceIdr: "", stock: "" });
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!form.name.trim() || form.priceIdr === "") {
      setError("Give the product a name and a price.");
      return;
    }
    const price = Number(form.priceIdr);
    const stock = Number(form.stock || "0");
    setBusy(true);
    setError(null);
    try {
      const created = await authRequest<Product>("/products", {
        method: "POST",
        body: { name: form.name.trim(), category: form.category, description: form.description.trim(), priceIdr: price, imageUrl: "" },
      });
      await authRequest(`/inventory/${created.id}`, { method: "PUT", body: { quantity: stock } });
      setForm({ name: "", category: "pastry", description: "", priceIdr: "", stock: "" });
      await onAdded(created.name);
    } catch {
      setError("The product could not be added. Try again.");
    } finally {
      setBusy(false);
    }
  };

  const field = "min-h-10 rounded-[10px] border border-field bg-white px-2.5";
  return (
    <form onSubmit={submit} className="flex flex-wrap items-end gap-3 border-t border-line px-6 py-5" aria-labelledby="add-title">
      <h3 id="add-title" className="w-full font-semibold">
        Add a product <span className="font-normal text-muted">(it shows a drawing until you add a photo path)</span>
      </h3>
      <label className="flex flex-col gap-1 text-sm">
        Name
        <input className={field} value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} maxLength={120} />
      </label>
      <label className="flex flex-col gap-1 text-sm">
        Category
        <select className={field} value={form.category} onChange={(e) => setForm({ ...form, category: e.target.value as Category })}>
          <option value="pastry">Pastry</option>
          <option value="flower">Flowers</option>
        </select>
      </label>
      <label className="flex grow flex-col gap-1 text-sm">
        Description
        <input className={field} value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })} maxLength={1000} />
      </label>
      <div className="flex flex-col gap-1 text-sm">
        <label htmlFor="new-price">Price</label>
        <MoneyInput id="new-price" value={form.priceIdr} onChange={(v) => setForm({ ...form, priceIdr: v })} />
      </div>
      <label className="flex flex-col gap-1 text-sm">
        Stock
        <input
          className={`${field} w-20 tabular-nums`}
          inputMode="numeric"
          maxLength={6}
          placeholder="0"
          value={form.stock}
          onChange={(e) => setForm({ ...form, stock: digitsOnly(e.target.value) })}
        />
      </label>
      <button type="submit" disabled={busy} className="min-h-11 cursor-pointer rounded-full bg-forest px-5 font-semibold text-white disabled:opacity-70">
        {busy ? "Adding…" : "Add product"}
      </button>
      {error && (
        <p role="alert" className="w-full text-sm text-rose-deep">
          {error}
        </p>
      )}
    </form>
  );
}
