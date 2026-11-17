import Image from "next/image";
import { Reveal } from "../Reveal";

// Replace the bracketed placeholders with the shop's real details.
export function Visit() {
  return (
    <section id="visit" aria-labelledby="visit-title" className="bg-sand">
      <div className="mx-auto flex max-w-[1180px] flex-wrap items-center gap-10 px-6 py-14">
        <Reveal className="min-w-0 flex-[1_1_420px]">
          <p className="font-hand text-[26px] text-rose">come say hello</p>
          <h2 id="visit-title" className="font-serif text-[40px] font-bold">
            Visit the shop
          </h2>
          <dl className="mt-5 grid grid-cols-[auto_1fr] gap-x-5 gap-y-2.5">
            <dt className="font-bold">Address</dt>
            <dd className="text-muted">[YOUR ADDRESS]</dd>
            <dt className="font-bold">Opening hours</dt>
            <dd className="text-muted">[OPENING HOURS]</dd>
            <dt className="font-bold">Contact</dt>
            <dd className="text-muted">[PHONE OR WHATSAPP]</dd>
          </dl>
          <p className="mt-5 text-muted">Online orders are open any time.</p>
        </Reveal>
        <Reveal delay={0.1} className="min-w-0 flex-[1_1_420px]">
          <Image
            src="/images/store/florist-at-work.webp"
            alt="A florist working at a table inside a flower shop"
            width={1400}
            height={1000}
            sizes="(max-width: 1000px) 100vw, 560px"
            className="aspect-[7/5] w-full rounded-[28px] object-cover"
          />
        </Reveal>
      </div>
    </section>
  );
}
