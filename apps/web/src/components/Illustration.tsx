"use client";

import { motion } from "motion/react";
import type { IllustrationKind } from "@/lib/illustrations";

const ink = "#3A2A20";
const leaf = "#5E9C72";
const stem = "#2F6B4A";
const line = { stroke: ink, strokeWidth: 2.5, strokeLinejoin: "round" as const, strokeLinecap: "round" as const };

const labels: Record<IllustrationKind, string> = {
  croissant: "Croissant illustration",
  eclair: "Eclair illustration",
  tart: "Strawberry tart illustration",
  rose: "Rose illustration",
  sunflower: "Sunflower illustration",
  lily: "Lily in a vase illustration",
  sprig: "Leafy sprig",
};

const petal = "M46 38 C 41 30, 43 20, 46 16 C 49 20, 51 30, 46 38 Z";
const lilyPetal = (cx: number, cy: number) => `M${cx} ${cy} C ${cx - 5} ${cy - 8}, ${cx - 3} ${cy - 18}, ${cx} ${cy - 22} C ${cx + 3} ${cy - 18}, ${cx + 5} ${cy - 8}, ${cx} ${cy} Z`;
const leafShape = "M0 0 C 6 -6, 16 -6, 20 0 C 16 6, 6 6, 0 0 Z";

const drawings: Record<IllustrationKind, React.ReactNode> = {
  croissant: (
    <>
      <ellipse cx="60" cy="92" rx="40" ry="5" fill="#2B2622" opacity="0.08" />
      <g {...line}>
        <path d="M14 80 C 18 46, 44 30, 60 30 C 76 30, 102 46, 106 80 C 96 72, 84 70, 76 74 C 70 68, 50 68, 44 74 C 36 70, 24 72, 14 80 Z" fill="#EDB75A" />
        <path d="M44 74 C 42 56, 48 40, 56 32" fill="none" />
        <path d="M76 74 C 78 56, 72 40, 64 32" fill="none" />
        <path d="M28 76 C 28 62, 32 52, 40 44" fill="none" />
        <path d="M92 76 C 92 62, 88 52, 80 44" fill="none" />
      </g>
      <path d="M50 40 C 54 37, 60 36, 66 37" stroke="#FBE3B0" strokeWidth="3" fill="none" strokeLinecap="round" />
    </>
  ),
  eclair: (
    <>
      <ellipse cx="60" cy="88" rx="44" ry="5" fill="#2B2622" opacity="0.08" />
      <g {...line}>
        <rect x="16" y="50" width="88" height="30" rx="15" fill="#F1D29A" />
        <path d="M20 60 C 22 50, 30 48, 40 49 L 82 49 C 92 48, 100 52, 100 60 C 92 56, 84 58, 76 56 C 66 59, 56 55, 46 58 C 36 55, 28 59, 20 60 Z" fill="#5B3A29" />
      </g>
      <path d="M34 53 l5 0 M56 52 l7 0 M78 52 l5 0" stroke="#F6E2C0" strokeWidth="2.5" strokeLinecap="round" />
    </>
  ),
  tart: (
    <>
      <ellipse cx="60" cy="94" rx="42" ry="5" fill="#2B2622" opacity="0.08" />
      <g {...line}>
        <path d="M18 66 L 26 86 C 40 92, 80 92, 94 86 L 102 66 Z" fill="#E2B270" />
        <path d="M30 70 L 33 84 M46 72 L 47 88 M60 72 L 60 89 M74 72 L 73 88 M90 70 L 87 84" fill="none" opacity="0.5" />
        <ellipse cx="60" cy="66" rx="42" ry="12" fill="#F7E6C4" />
        <circle cx="44" cy="63" r="7" fill="#D2404F" />
        <circle cx="62" cy="59" r="7" fill="#D2404F" />
        <circle cx="78" cy="64" r="7" fill="#D2404F" />
        <circle cx="58" cy="70" r="6" fill="#D2404F" />
      </g>
      <path d="M44 56 l-3 -4 M62 52 l3 -4 M78 57 l-2 -4" stroke={stem} strokeWidth="2.5" strokeLinecap="round" />
    </>
  ),
  rose: (
    <g {...line}>
      <path d="M60 64 C 58 80, 62 94, 60 112" fill="none" stroke={stem} strokeWidth="3" />
      <path d="M60 86 C 48 78, 38 82, 32 88 C 42 94, 54 93, 60 86 Z" fill={leaf} />
      <path d="M61 98 C 72 90, 82 92, 88 98 C 78 104, 67 104, 61 98 Z" fill={leaf} />
      <path d="M38 44 C 36 28, 50 18, 60 20 C 72 18, 86 28, 82 46 C 80 58, 70 66, 60 66 C 48 66, 40 58, 38 44 Z" fill="#E06C88" />
      <path d="M60 30 C 50 30, 48 42, 56 46 C 64 50, 70 42, 66 36" fill="none" />
      <path d="M46 42 C 46 54, 56 60, 66 56 C 74 52, 77 44, 75 37" fill="none" />
    </g>
  ),
  sunflower: (
    <>
      <g {...line}>
        <path d="M60 60 C 62 78, 58 94, 60 112" fill="none" stroke={stem} strokeWidth="3" />
        <path d="M60 92 C 70 82, 82 82, 90 88 C 82 96, 70 98, 60 92 Z" fill={leaf} />
        <g fill="#F0BE3C">
          {Array.from({ length: 12 }, (_, i) => (
            <ellipse key={i} cx="60" cy="24" rx="7" ry="13" transform={`rotate(${i * 30} 60 44)`} />
          ))}
        </g>
        <circle cx="60" cy="44" r="14" fill="#6B4528" />
      </g>
      <g fill={ink}>
        <circle cx="55" cy="40" r="1.6" />
        <circle cx="62" cy="38" r="1.6" />
        <circle cx="66" cy="45" r="1.6" />
        <circle cx="58" cy="48" r="1.6" />
        <circle cx="53" cy="46" r="1.6" />
      </g>
    </>
  ),
  lily: (
    <>
      <g {...line}>
        <path d="M56 76 C 54 64, 50 52, 46 40" fill="none" stroke={stem} strokeWidth="3" />
        <path d="M64 76 C 66 62, 70 50, 76 36" fill="none" stroke={stem} strokeWidth="3" />
        <path d="M52 62 C 42 58, 36 50, 36 44 C 44 46, 50 52, 52 62 Z" fill={leaf} />
        <g fill="#FFFFFF">
          {[0, 72, 144, 216, 288].map((a) => (
            <path key={`a${a}`} d={petal} transform={`rotate(${a} 46 38)`} />
          ))}
          {[0, 72, 144, 216, 288].map((a) => (
            <path key={`b${a}`} d={lilyPetal(76, 34)} transform={`rotate(${a} 76 34)`} />
          ))}
        </g>
        <path d="M42 74 L 78 74 L 73 106 C 67 111, 53 111, 47 106 Z" fill="#DCEAE2" />
        <path d="M45 84 L 75 84" fill="none" opacity="0.4" />
      </g>
      <circle cx="46" cy="38" r="3" fill="#F0BE3C" />
      <circle cx="76" cy="34" r="3" fill="#F0BE3C" />
    </>
  ),
  sprig: (
    <>
      <path d="M14 104 C 40 84, 70 54, 104 18" fill="none" stroke={stem} strokeWidth="2.5" strokeLinecap="round" />
      <g fill="#7DB08E" stroke={stem} strokeWidth="2" strokeLinejoin="round">
        {[
          [28, 92, -100],
          [40, 81, 10],
          [55, 67, -100],
          [67, 55, 10],
          [81, 40, -100],
          [92, 29, 10],
        ].map(([x, y, r]) => (
          <path key={`${x}-${y}`} d={leafShape} transform={`translate(${x} ${y}) rotate(${r})`} />
        ))}
      </g>
      <circle cx="104" cy="18" r="4" fill="#E06C88" />
      <circle cx="96" cy="14" r="3" fill="#E06C88" />
    </>
  ),
};

type Props = {
  kind: IllustrationKind;
  size?: number;
  /** Decorative drawings are hidden from screen readers. */
  decorative?: boolean;
  /** A slow, gentle sway (turned off for people who prefer reduced motion). */
  sway?: boolean;
  className?: string;
};

export function Illustration({ kind, size = 120, decorative = false, sway = false, className }: Props) {
  const a11y = decorative ? { "aria-hidden": true } : { role: "img", "aria-label": labels[kind] };
  return (
    <motion.svg
      viewBox="0 0 120 120"
      width={size}
      height={size}
      className={className}
      style={{ transformOrigin: "50% 90%", flexShrink: 0 }}
      animate={sway ? { rotate: [-3, 3, -3] } : undefined}
      transition={sway ? { duration: 6, repeat: Infinity, ease: "easeInOut" } : undefined}
      {...a11y}
    >
      {drawings[kind]}
    </motion.svg>
  );
}
