/** Indonesian rupiah with dot thousands separators: 28000 -> "Rp 28.000". */
export function rupiah(amount: number): string {
  return "Rp " + Math.round(amount).toString().replace(/\B(?=(\d{3})+(?!\d))/g, ".");
}

export function stockLabel(stock: number | null): string {
  if (stock === null) return "Not stocked yet";
  if (stock === 0) return "Sold out";
  if (stock <= 5) return `Only ${stock} left`;
  return `${stock} in stock`;
}

export function shortId(id: string): string {
  return id.slice(0, 8);
}

export function orderDate(iso: string): string {
  return new Intl.DateTimeFormat("en-GB", { dateStyle: "medium", timeStyle: "short" }).format(new Date(iso));
}
