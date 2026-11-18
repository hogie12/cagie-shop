import type { Metadata } from "next";

// Orders are private: keep them out of search engines. Also covers /orders/[id].
export const metadata: Metadata = { title: "My orders", robots: { index: false, follow: false } };

export default function OrdersLayout({ children }: LayoutProps<"/orders">) {
  return children;
}
