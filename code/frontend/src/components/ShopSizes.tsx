import { T } from "../editable";
import { sizeKey } from "../lib/shop";
import { sizeColumns } from "./SizePicker";

/**
 * The Size (EU) group of the shop's rail: one pill per size a live shoe
 * lists, pressed when it is in the address. Every pill is a button with
 * `aria-pressed`, so the choice is read without colour. Fewer than two sizes
 * is nothing to choose between, so the group — heading and pills — is not
 * drawn (as FilterRail does for its own groups); a size already in the address
 * still filters and still counts.
 */
export default function ShopSizes({ sizes, pressed, onToggle }: {
  sizes: readonly string[];
  pressed: readonly string[];
  onToggle: (size: string) => void;
}) {
  if (sizes.length < 2) return null;
  return (
    <div className="mt-6 border-b border-line pb-5">
      <T k="pages.shop.filter_size" as="p" className="pb-3 text-xs uppercase tracking-wider text-ink-soft" />
      <div className={`grid gap-1.5 ${sizeColumns(sizes.length, 4)}`}>
        {sizes.map((s) => {
          const on = pressed.includes(s);
          return (
            <button key={s} type="button" aria-pressed={on} onClick={() => onToggle(s)} data-testid={`filter-size-${sizeKey(s)}`}
              className={`whitespace-nowrap rounded-[var(--radius-pill)] border px-1 py-1.5 text-[0.8125rem] tabular-nums ${on ? "border-ink bg-ink text-ground" : "border-line text-ink hover:border-ink"}`}>
              {s}
            </button>
          );
        })}
      </div>
    </div>
  );
}
