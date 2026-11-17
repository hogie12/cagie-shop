import { Illustration } from "../Illustration";
import { Reveal } from "../Reveal";
import type { IllustrationKind } from "@/lib/illustrations";

const steps: { word: string; kind: IllustrationKind; title: string; text: string }[] = [
  { word: "one", kind: "croissant", title: "Pick your favourites", text: "Mix pastries and flowers in one cart. Stock is shown on every item." },
  { word: "two", kind: "rose", title: "Reserved and paid in seconds", text: "Your items are held for you before any payment is taken." },
  { word: "three", kind: "sunflower", title: "Follow it live", text: "Your order page updates by itself, from placed to confirmed." },
];

export function HowItWorks() {
  return (
    <section aria-labelledby="how-title" className="bg-sand">
      <div className="mx-auto max-w-[1180px] px-6 py-12">
        <Reveal>
          <h2 id="how-title" className="mb-6 font-serif text-[32px] font-bold">
            How ordering works
          </h2>
        </Reveal>
        <ol className="grid list-none grid-cols-[repeat(auto-fit,minmax(260px,1fr))] gap-5 p-0">
          {steps.map((s, i) => (
            <li key={s.word}>
              <Reveal delay={i * 0.1} className="flex h-full items-start gap-4 rounded-[22px] bg-white p-5">
                <Illustration kind={s.kind} size={64} decorative />
                <div>
                  <p className="font-hand text-2xl text-rose">{s.word}</p>
                  <h3 className="text-lg font-bold">{s.title}</h3>
                  <p className="mt-1 text-sm text-muted">{s.text}</p>
                </div>
              </Reveal>
            </li>
          ))}
        </ol>
      </div>
    </section>
  );
}
