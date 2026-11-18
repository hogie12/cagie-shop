/** The Cagie mark: a croissant cradling a rose bud, pastry and flowers in one
 * shape. Same drawing as src/app/icon.svg (the favicon). */
export function LogoMark({ size = 40 }: { size?: number }) {
  return (
    <svg viewBox="0 0 64 64" width={size} height={size} aria-hidden="true" className="shrink-0">
      <rect width="64" height="64" rx="14" fill="#1F4D3A" />
      <g fill="#EDB75A" stroke="#C48A32" strokeWidth="1.6">
        <ellipse cx="13.5" cy="32" rx="5" ry="3.8" transform="rotate(-38 13.5 32)" />
        <ellipse cx="50.5" cy="32" rx="5" ry="3.8" transform="rotate(38 50.5 32)" />
        <ellipse cx="21" cy="38" rx="7.5" ry="5.8" transform="rotate(-26 21 38)" />
        <ellipse cx="43" cy="38" rx="7.5" ry="5.8" transform="rotate(26 43 38)" />
        <ellipse cx="32" cy="41" rx="9.5" ry="7" />
      </g>
      <circle cx="32" cy="22" r="9.5" fill="#E06C88" />
      <path d="M32 17.2 C 27.7 17.2, 26.8 23, 30.6 24.4 C 34.4 25.8, 36.8 22, 34.9 19.6" stroke="#FAF6F0" strokeWidth="2" strokeLinecap="round" fill="none" />
      <path d="M39.5 15 C 43.5 11, 48.5 12, 49.5 14 C 46.5 18, 42.5 18, 39.5 15 Z" fill="#7DB08E" />
    </svg>
  );
}

/** Mark plus wordmark, as used in the header. */
export function Logo() {
  return (
    <span className="flex items-center gap-2.5">
      <LogoMark size={40} />
      <span className="flex flex-col leading-none">
        <span className="font-serif text-[28px] font-bold text-forest">Cagie</span>
        {/* Hidden on the smallest phones so the header stays one row high. */}
        <span className="font-hand text-lg text-rose max-[359px]:hidden">pastry &amp; flowers</span>
      </span>
    </span>
  );
}
