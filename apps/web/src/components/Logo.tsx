/** The Cagie mark: a strawberry tart seen from above whose berries form a
 * flower, pastry and flowers in one shape. Same drawing as src/app/icon.svg
 * (the favicon); the alternatives are in design/logo. */
export function LogoMark({ size = 40 }: { size?: number }) {
  return (
    <svg viewBox="0 0 64 64" width={size} height={size} aria-hidden="true" className="shrink-0">
      <rect width="64" height="64" rx="14" fill="#F7EBD3" />
      <circle cx="32" cy="32" r="25" fill="#E2B270" />
      <circle cx="32" cy="32" r="20" fill="#F7E6C4" />
      <g fill="#D2404F">
        {[0, 60, 120, 180, 240, 300].map((a) => (
          <ellipse key={a} cx="32" cy="19" rx="6" ry="8" transform={`rotate(${a} 32 32)`} />
        ))}
      </g>
      <circle cx="32" cy="32" r="5.5" fill="#2F6B4A" />
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
