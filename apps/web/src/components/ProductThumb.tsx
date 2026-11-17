import Image from "next/image";
import { illustrationFor } from "@/lib/illustrations";
import type { Category } from "@/lib/types";
import { Illustration } from "./Illustration";

/** A small square product picture: the photo, or its drawing when there is none. */
export function ProductThumb({ name, category, imageUrl, size }: { name: string; category: Category; imageUrl: string; size: number }) {
  if (imageUrl) {
    return <Image src={imageUrl} alt="" width={size} height={size} className="shrink-0 rounded-2xl object-cover" style={{ width: size, height: size }} />;
  }
  return (
    <span className="flex shrink-0 items-center justify-center rounded-2xl bg-butter" style={{ width: size, height: size }}>
      <Illustration kind={illustrationFor(name, category)} size={size * 0.8} decorative />
    </span>
  );
}
