import type { Metadata } from "next";

// The cart page is a client component, so its title lives here.
export const metadata: Metadata = { title: "Your cart", robots: { index: false, follow: false } };

export default function CartLayout({ children }: LayoutProps<"/cart">) {
  return children;
}
