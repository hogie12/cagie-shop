import type { OrderStatus } from "@/lib/types";

const styles: Record<OrderStatus, [string, string]> = {
  pending: ["Checking stock", "bg-sun text-[#6B4E26]"],
  awaiting_payment: ["Taking payment", "bg-sun text-[#6B4E26]"],
  confirmed: ["Confirmed", "bg-forest-tint text-forest"],
  cancelled: ["Cancelled", "bg-rose-tint text-rose-dark"],
};

export function StatusPill({ status }: { status: OrderStatus }) {
  const [label, cls] = styles[status];
  return (
    <span className={`inline-flex items-center gap-1.5 rounded-full px-3 py-1 text-[13px] font-semibold ${cls}`}>
      <span aria-hidden="true" className="h-2 w-2 rounded-full bg-current" />
      {label}
    </span>
  );
}
