import type { MetadataRoute } from "next";

const site = process.env.SITE_URL ?? "https://cagie-shop.hogie.my.id";

// The home page is the only public page; products live on it.
export default function sitemap(): MetadataRoute.Sitemap {
  return [{ url: site, changeFrequency: "daily", priority: 1 }];
}
