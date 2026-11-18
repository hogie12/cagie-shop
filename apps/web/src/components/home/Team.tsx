import Image from "next/image";
import { Reveal } from "../Reveal";

type Person = {
  name: string;
  role: string;
  roleColor: string;
  photo: { src: string; alt: string; position: string; tint: string };
  bio: string;
  favouriteLabel: string;
  favourite: string;
  /** Sits lower than the first on wide screens so the pair does not read as twin cards. */
  offset?: boolean;
};

const people: Person[] = [
  {
    name: "Hogie",
    role: "the baker",
    roleColor: "text-forest",
    photo: {
      src: "/images/team/baker.jpg",
      alt: "Hogie, the Cagie baker, in sunglasses and a plum sweatshirt, one hand raised to the hair",
      position: "50% 30%",
      tint: "bg-butter",
    },
    bio: "Hogie starts before the shop opens, when the ovens are the only warm thing in the building. The case stays short on purpose, so every croissant, eclair and tart in it was baked that morning.",
    favouriteLabel: "Favourite to bake:",
    favourite: "Strawberry Tart, because the custard has to be right",
  },
  {
    name: "Tasya",
    role: "the florist",
    roleColor: "text-rose",
    photo: {
      src: "/images/team/florist.jpg",
      alt: "Tasya, the Cagie florist, smiling, holding a wrapped bouquet of blue roses, pink chrysanthemums and white lilies",
      position: "50% 35%",
      tint: "bg-rose-tint",
    },
    bio: "Tasya picks flowers at the market and puts each bouquet together when you order it, wrapped in paper or set in a glass vase. Colour mixes you would not have thought to try are welcome.",
    favouriteLabel: "Favourite to arrange:",
    favourite: "Tulips, for how they keep moving after they are cut",
    offset: true,
  },
];

export function Team() {
  return (
    <section id="team" aria-labelledby="team-title" className="mx-auto max-w-[1180px] px-6 pb-20 pt-[72px]">
      <Reveal className="max-w-[560px]">
        <h2 id="team-title" className="font-serif text-[40px] font-bold leading-[1.1]">
          Meet our <em className="font-semibold text-forest">baker</em> and <em className="font-semibold text-rose">florist</em>
        </h2>
        <p className="mt-4 text-base text-muted">Two people, one small shop. Everything you order from Cagie is made by hand, by one of them.</p>
      </Reveal>

      <div className="mt-12 flex flex-wrap items-start gap-x-14 gap-y-12">
        {people.map((p, i) => (
          <Reveal key={p.name} delay={i * 0.1} className={`min-w-0 flex-[1_1_440px] ${p.offset ? "min-[900px]:mt-16" : ""}`}>
            <article>
              <div className={`overflow-hidden rounded-[28px] ${p.photo.tint}`}>
                <Image
                  src={p.photo.src}
                  alt={p.photo.alt}
                  width={900}
                  height={1200}
                  sizes="(max-width: 1000px) 100vw, 560px"
                  className="aspect-[4/5] max-h-[560px] w-full object-cover"
                  style={{ objectPosition: p.photo.position }}
                />
              </div>
              <div className="mt-5 flex flex-wrap items-baseline gap-x-3.5 gap-y-1">
                <h3 className="font-serif text-[30px] font-bold leading-[1.15]">{p.name}</h3>
                <span className={`font-hand text-[26px] font-bold ${p.roleColor}`}>{p.role}</span>
              </div>
              <p className="mt-2.5 max-w-[480px] text-base text-muted">{p.bio}</p>
              <p className="mt-3.5">
                <strong>{p.favouriteLabel}</strong> <span className="text-muted">{p.favourite}</span>
              </p>
            </article>
          </Reveal>
        ))}
      </div>
    </section>
  );
}
