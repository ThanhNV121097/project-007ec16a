import { type ReactNode } from "react";
import { Link } from "react-router-dom";
import { sizeKey } from "../lib/shop";

const SIZE_COLS: Record<number, string> = { 3: "grid-cols-3", 4: "grid-cols-4", 5: "grid-cols-4 sm:grid-cols-5" };

/** The column count that leaves no lone pill in the last row: 9 → 5 + 4, 8 → 4 + 4, 6 → 3 + 3. */
export function sizeColumns(count: number, max = 5): string {
  for (let c = max; c >= 3; c--) {
    const rest = count % c;
    if (rest === 0 || rest >= c - 1) return SIZE_COLS[c];
  }
  return SIZE_COLS[4];
}

/**
 * THE SIZES, WITH THE STOCK IN THEM. Every EU size the pair is made in, as
 * pills that hold the choice: a sold-out size is struck through and cannot
 * be chosen, a size with its last pairs carries a small dot, and the size
 * chart is one tap away on the label's own line.
 */
export default function SizePicker({ sizes, low = [], out = [], value, onChange, label, chart }: {
  sizes: readonly string[];
  low?: readonly string[];
  out?: readonly string[];
  value: string;
  onChange: (size: string) => void;
  label: ReactNode;
  chart: ReactNode;
}) {
  if (sizes.length === 0) return null;
  return (
    <div data-block="variants" className="flex flex-col gap-3">
      <div className="flex items-baseline justify-between gap-4">
        <p id="size-label" className="text-xs uppercase tracking-wider text-ink-soft">{label}</p>
        <Link to="/size-chart" className="text-xs text-ink underline underline-offset-4 hover:no-underline">{chart}</Link>
      </div>
      <div role="radiogroup" aria-labelledby="size-label" className={`grid gap-2 ${sizeColumns(sizes.length)}`}>
        {sizes.map((s) => {
          const on = s === value;
          const gone = out.includes(s);
          const last = low.includes(s);
          return (
            <button
              key={s}
              type="button"
              role="radio"
              aria-checked={on}
              disabled={gone}
              data-testid={`size-${sizeKey(s)}`}
              onClick={() => onChange(s)}
              className={`relative whitespace-nowrap rounded-[var(--radius-pill)] border px-2 py-2.5 text-sm tabular-nums transition-colors disabled:cursor-not-allowed disabled:border-line disabled:text-ink-soft disabled:line-through ${on ? "border-ink bg-ink text-ground" : "border-line text-ink hover:border-ink"}`}
            >
              {s}
              {last && !gone && (
                <span aria-hidden="true" className={`absolute right-2 top-1.5 size-1.5 rounded-full ${on ? "bg-ground" : "bg-danger"}`} />
              )}
            </button>
          );
        })}
      </div>
    </div>
  );
}
