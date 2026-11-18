"use client";

import { useLayoutEffect, useRef } from "react";
import { digitsOnly, groupThousands } from "@/lib/format";

type Props = {
  /** Plain digits, e.g. "28000". The field shows "28.000". */
  value: string;
  onChange: (digits: string) => void;
  id?: string;
  "aria-label"?: string;
  className?: string;
};

/** A rupiah field that adds thousand separators while typing and keeps the
 * caret next to the digit the person just typed. */
export function MoneyInput({ value, onChange, id, className = "w-[140px]", ...aria }: Props) {
  const ref = useRef<HTMLInputElement>(null);
  // How many digits sat left of the caret after the last edit.
  const digitsBeforeCaret = useRef<number | null>(null);

  useLayoutEffect(() => {
    const el = ref.current;
    const target = digitsBeforeCaret.current;
    if (!el || target === null || document.activeElement !== el) return;
    let pos = 0;
    for (let seen = 0; pos < el.value.length && seen < target; pos++) {
      if (/\d/.test(el.value[pos])) seen++;
    }
    el.setSelectionRange(pos, pos);
    digitsBeforeCaret.current = null;
  });

  return (
    <span
      className={`flex min-h-10 items-center rounded-[10px] border border-field bg-white focus-within:outline-2 focus-within:outline-offset-2 focus-within:outline-forest ${className}`}
    >
      <span aria-hidden="true" className="pl-2.5 text-sm text-muted">
        Rp
      </span>
      <input
        ref={ref}
        id={id}
        inputMode="numeric"
        autoComplete="off"
        value={groupThousands(value)}
        onChange={(e) => {
          const el = e.target;
          digitsBeforeCaret.current = el.value.slice(0, el.selectionStart ?? el.value.length).replace(/\D/g, "").length;
          onChange(digitsOnly(el.value).slice(0, 12));
        }}
        className="min-w-0 flex-1 bg-transparent px-1.5 py-2 tabular-nums focus-visible:outline-none"
        {...aria}
      />
    </span>
  );
}
