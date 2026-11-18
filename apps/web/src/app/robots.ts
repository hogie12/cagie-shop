import type { MetadataRoute } from "next";

const site = process.env.SITE_URL ?? "https://cagie-shop.hogie.my.id";

// Search engines may read the shop; private pages stay out.
export default function robots(): MetadataRoute.Robots {
  return {
    rules: { userAgent: "*", allow: "/", disallow: ["/admin", "/cart", "/orders", "/login"] },
    sitemap: `${site}/sitemap.xml`,
  };
}
