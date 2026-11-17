import { Illustration } from "./Illustration";

export function SiteFooter() {
  return (
    <footer className="border-t border-line bg-ground">
      <div className="mx-auto flex max-w-[1180px] flex-wrap items-center justify-between gap-3 px-6 py-6 text-[13px] text-muted">
        <span className="flex items-center gap-2">
          <Illustration kind="sprig" size={28} decorative />
          Cagie Pastry &amp; Flower Shop
        </span>
        <span>
          Photos from{" "}
          <a href="https://unsplash.com" className="underline" target="_blank" rel="noreferrer">
            Unsplash
          </a>
        </span>
      </div>
    </footer>
  );
}
