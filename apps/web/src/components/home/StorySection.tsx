import Image from "next/image";
import { Illustration } from "../Illustration";
import { Reveal } from "../Reveal";
import { illustrationFor } from "@/lib/illustrations";
import type { ProductWithStock } from "@/lib/types";

type Props = {
  id?: string;
  kicker: string;
  title: string;
  emphasis: string;
  text: string;
  image: { src: string; alt: string };
  products: ProductWithStock[];
  /** Photo on the right instead of the left (on wide screens). */
  reversed?: boolean;
};

export function StorySection({ id, kicker, title, emphasis, text, image, products, reversed }: Props) {
  const headingId = `${kicker.replace(/\s+/g, "-")}-title`;
  return (
    <section id={id} aria-labelledby={headingId} className={`mx-auto flex max-w-[1180px] flex-wrap items-center gap-10 px-6 py-12 ${reversed ? "flex-row-reverse" : ""}`}>
      <Reveal className="min-w-0 flex-[1_1_440px]">
        <Image src={image.src} alt={image.alt} width={1400} height={1000} sizes="(max-width: 1000px) 100vw, 560px" className="aspect-[7/5] w-full rounded-[28px] object-cover" />
      </Reveal>
      <Reveal delay={0.1} className="min-w-0 flex-[1_1_440px]">
        <p className="font-hand text-[26px] text-rose">{kicker}</p>
        <h2 id={headingId} className="font-serif text-[40px] font-bold leading-[1.1]">
          {title} <em className="font-semibold">{emphasis}</em>
        </h2>
        <p className="mt-4 text-base text-muted">{text}</p>
        <ul className="mt-5 flex list-none flex-col gap-3 p-0">
          {products.map((p) => (
            <li key={p.id} className="flex items-center gap-3">
              <Illustration kind={illustrationFor(p.name, p.category)} size={40} decorative />
              <span>
                <strong>{p.name}</strong> <span className="text-muted">{p.description.toLowerCase()}</span>
              </span>
            </li>
          ))}
        </ul>
      </Reveal>
    </section>
  );
}
