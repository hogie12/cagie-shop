import type { Order } from "./types";

export type StepState = "done" | "current" | "failed" | "todo";

export type OrderView = {
  headline: string;
  subline: string;
  steps: { label: string; note: string; state: StepState }[];
  live: boolean;
  confirmed: boolean;
  cancelled: boolean;
  /** Which step failed when cancelled: stock (step 2) or payment (step 3). */
  failedAt: "stock" | "payment" | null;
};

/** Turns an order's status into what the order page shows. Mirrors the saga:
 * pending (stock) -> awaiting_payment -> confirmed, or cancelled at either step. */
export function describeOrder(order: Pick<Order, "status" | "cancelReason">): OrderView {
  const reason = order.cancelReason ?? "";
  const failedAt = order.status !== "cancelled" ? null : reason.startsWith("payment failed") ? "payment" : "stock";

  const states: Record<string, StepState[]> = {
    pending: ["done", "current", "todo", "todo"],
    awaiting_payment: ["done", "done", "current", "todo"],
    confirmed: ["done", "done", "done", "done"],
    stock: ["done", "failed", "todo", "todo"],
    payment: ["done", "done", "failed", "todo"],
  };
  const s = states[failedAt ?? order.status];

  const notes = [
    "Received by Cagie",
    failedAt === "stock" ? "Not enough stock" : failedAt === "payment" ? "Released again" : "Items held for you",
    failedAt === "payment" ? "Not completed" : "Card charged",
    "All done",
  ];
  const labels = ["Order placed", "Stock reserved", "Payment", "Confirmed"];

  const copy: Record<string, [string, string]> = {
    pending: ["Checking stock", "Hold on, this usually takes a few seconds."],
    awaiting_payment: ["Taking payment", "Your items are reserved. Cagie is charging your card now."],
    confirmed: ["Order confirmed", "Your items are reserved and paid for."],
    cancelled: ["Order cancelled", "This order could not be completed. Nothing was charged."],
  };
  const [headline, subline] = copy[order.status];

  return {
    headline,
    subline,
    steps: labels.map((label, i) => ({ label, note: s[i] === "todo" ? "Waiting" : notes[i], state: s[i] })),
    live: order.status === "pending" || order.status === "awaiting_payment",
    confirmed: order.status === "confirmed",
    cancelled: order.status === "cancelled",
    failedAt,
  };
}

/** A cancel reason from the backend, rewritten for a customer. */
export function friendlyReason(reason: string | undefined): string {
  if (!reason) return "The order could not be completed.";
  if (reason.startsWith("payment failed: ")) {
    const why = reason.slice("payment failed: ".length);
    return `Payment failed: ${why}. Your items are back on the shelf.`;
  }
  if (reason.startsWith("insufficient stock")) return "Not enough stock for one of your items. Try a smaller quantity.";
  if (reason.includes("is not stocked")) return "One of your items is not in stock at the moment.";
  return reason;
}
