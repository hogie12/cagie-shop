import { GiftPairing } from "@/components/home/GiftPairing";
import { Hero } from "@/components/home/Hero";
import { HowItWorks } from "@/components/home/HowItWorks";
import { StorySection } from "@/components/home/StorySection";
import { Visit } from "@/components/home/Visit";
import { Illustration } from "@/components/Illustration";
import { ProductGrid } from "@/components/ProductGrid";
import { getCatalog } from "@/lib/catalog";

export default async function Home() {
  const products = await getCatalog();
  const inStock = (products ?? []).filter((p) => (p.stock ?? 0) > 0);
  const lowest = [...inStock].sort((a, b) => (a.stock ?? 0) - (b.stock ?? 0))[0];
  const spotlight = lowest && (lowest.stock ?? 0) <= 5 ? lowest : null;
  const pastries = (products ?? []).filter((p) => p.category === "pastry");
  const flowers = (products ?? []).filter((p) => p.category === "flower");
  const giftFlower = flowers.find((p) => /rose/i.test(p.name)) ?? flowers[0];
  const giftPastry = pastries.find((p) => /tart/i.test(p.name)) ?? pastries[0];

  return (
    <main>
      <Hero spotlight={spotlight} />
      <HowItWorks />

      <section id="products" aria-labelledby="products-title" className="mx-auto w-full max-w-[1180px] px-6 pb-6 pt-14">
        {products ? (
          <>
            <ProductGrid products={products} />
            <p className="mt-4 text-sm text-muted">Prices in rupiah, confirmed by the shop when you order.</p>
          </>
        ) : (
          <div className="flex flex-col items-center gap-3 rounded-3xl border border-line bg-white p-10 text-center">
            <Illustration kind="sprig" size={80} decorative />
            <h2 id="products-title" className="font-serif text-2xl font-bold">
              The shop is taking a short break
            </h2>
            <p className="text-muted">Products cannot be loaded right now. Please try again in a minute.</p>
          </div>
        )}
      </section>

      <div id="story">
        <StorySection
          title="A small case,"
          emphasis="baked every morning"
          text="Cagie keeps the pastry case short on purpose: a few things, made well and baked fresh each day. When today's batch is gone, it is gone."
          image={{ src: "/images/store/bakery-counter.webp", alt: "A display case filled with pastries" }}
          products={pastries}
        />
        <StorySection
          title="Flowers,"
          emphasis="arranged to order"
          text="Every bouquet is put together when you order it, wrapped in kraft paper or set in a glass vase, ready to give or to keep."
          image={{ src: "/images/store/flower-buckets.webp", alt: "Buckets of pink and white flowers outside a florist" }}
          products={flowers}
          reversed
        />
      </div>

      {giftFlower && giftPastry && <GiftPairing flower={giftFlower} pastry={giftPastry} />}
      <Visit />
    </main>
  );
}
