import type { Metadata, Viewport } from "next";
import { Caveat, Figtree, Fraunces } from "next/font/google";
import { Providers } from "@/components/Providers";
import { SiteFooter } from "@/components/SiteFooter";
import { SiteHeader } from "@/components/SiteHeader";
import "./globals.css";

const fraunces = Fraunces({ subsets: ["latin"], axes: ["opsz"], style: ["normal", "italic"], variable: "--font-fraunces", display: "swap" });
const figtree = Figtree({ subsets: ["latin"], variable: "--font-figtree", display: "swap" });
const caveat = Caveat({ subsets: ["latin"], variable: "--font-caveat", display: "swap" });

const description =
  "Butter croissants, eclairs and strawberry tarts, plus roses, sunflowers and lilies arranged to order. Order online and follow your order live.";

// Link previews need absolute URLs; set SITE_URL on another domain.
// The preview images come from opengraph-image.png and twitter-image.png.
export const metadata: Metadata = {
  metadataBase: new URL(process.env.SITE_URL ?? "https://cagie-shop.hogie.my.id"),
  title: { default: "Cagie | Pastries and flowers, ordered online", template: "%s | Cagie" },
  description,
  applicationName: "Cagie",
  openGraph: {
    type: "website",
    siteName: "Cagie",
    title: "Cagie | Pastries and flowers, ordered online",
    description,
    url: "/",
    locale: "en_US",
  },
  twitter: {
    card: "summary_large_image",
    title: "Cagie | Pastries and flowers, ordered online",
    description,
  },
};

// Colours the mobile browser bar to match the page.
export const viewport: Viewport = {
  themeColor: "#faf6f0",
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
