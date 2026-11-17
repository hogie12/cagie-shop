import type { Metadata } from "next";
import { Caveat, Figtree, Fraunces } from "next/font/google";
import { Providers } from "@/components/Providers";
import { SiteFooter } from "@/components/SiteFooter";
import { SiteHeader } from "@/components/SiteHeader";
import "./globals.css";

const fraunces = Fraunces({ subsets: ["latin"], axes: ["opsz"], style: ["normal", "italic"], variable: "--font-fraunces", display: "swap" });
const figtree = Figtree({ subsets: ["latin"], variable: "--font-figtree", display: "swap" });
const caveat = Caveat({ subsets: ["latin"], variable: "--font-caveat", display: "swap" });

export const metadata: Metadata = {
  title: { default: "Cagie Pastry & Flowers", template: "%s · Cagie" },
  description: "Fresh pastries and flowers arranged to order. Order online and follow your order live.",
};

export default function RootLayout({ children }: LayoutProps<"/">) {
  return (
    <html lang="en" className={`${fraunces.variable} ${figtree.variable} ${caveat.variable} h-full antialiased`}>
      <body className="flex min-h-full flex-col font-sans text-[15px] leading-normal">
        <Providers>
          <SiteHeader />
          <div className="flex grow flex-col">{children}</div>
          <SiteFooter />
        </Providers>
      </body>
    </html>
  );
}
