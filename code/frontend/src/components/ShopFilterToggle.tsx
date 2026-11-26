import { T } from "../editable";
import Icon from "./Icon";

/**
 * The phone's "Filter" button: below 1024px it opens and closes the rail, and
 * once anything is set it reads "Filter (3)" — the ticked boxes and the
 * pressed size pills, one each. From 1024px the rail is always there and the
 * button is not (`lg:hidden`).
 */
export default function ShopFilterToggle({ open, count, controls, onToggle }: {
  open: boolean;
  count: number;
  /** The id of the rail it opens. */
  controls: string;
  onToggle: () => void;
}) {
  return (
    <button type="button" onClick={onToggle} aria-expanded={open} aria-controls={controls}
      className="inline-flex items-center gap-2 rounded-[var(--radius-pill)] border border-ink px-4 py-2 text-[0.6875rem] font-medium uppercase tracking-[0.12em] text-ink lg:hidden">
      <Icon name="filter" size={14} />
      <span><T k="pages.shop.filters" />{count > 0 && <span className="tabular-nums">{` (${count})`}</span>}</span>
    </button>
  );
}
