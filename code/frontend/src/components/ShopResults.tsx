import { T } from "../editable";
import type { Line, Product } from "../lib/shop";
import Grid from "./Grid";
import Icon from "./Icon";
import ProductCard from "./ProductCard";
import ShopNotice, { PILL } from "./ShopNotice";

/**
 * WHAT THE SHOP'S RESULTS AREA SHOWS, one thing at a time and never a blank:
 * the first page on its way, the first page that could not come, no pair that
 * matches, or the shoes — with "Show more" under them, loading, or failed in
 * its place. Everything it knows comes in as props, so the page decides
 * nothing here and the backend step changes nothing here.
 *
 * `loading` with shoes on screen means the next page is on its way; `error`
 * with shoes on screen means the next page failed and the shoes stay.
 */
export interface ShopResultsProps {
  items: readonly Product[];
  /** The API's total, or null until the query's first answer is in. */
  total: number | null;
  hasMore: boolean;
  loading: boolean;
  error: Error | null;
  lines: ReadonlyMap<string, Line>;
  onMore: () => void;
  onRetry: () => void;
  onClear: () => void;
}

function Failed({ onRetry, className = "" }: { onRetry: () => void; className?: string }) {
  return (
    <ShopNotice className={className}>
      <div role="alert">
        <T k="pages.shop.error" as="h2" className="text-[length:var(--size-h2)] text-danger" />
      </div>
      <button type="button" onClick={onRetry} className={PILL}><T k="pages.shop.retry" /></button>
    </ShopNotice>
  );
}

function Empty({ onClear }: { onClear: () => void }) {
  return (
    <ShopNotice>
      <T k="pages.shop.empty" as="h2" className="text-[length:var(--size-h2)] text-ink" />
      <T k="pages.shop.empty_line" as="p" className="max-w-[44ch] text-sm text-ink-soft" />
      <button type="button" onClick={onClear} className={PILL}>
        <T k="pages.shop.clear" /><Icon name="arrow" size={14} />
      </button>
    </ShopNotice>
  );
}

function ShowMore({ loading, onMore }: { loading: boolean; onMore: () => void }) {
  return (
    <div className="mt-12 flex justify-center">
      <button type="button" onClick={onMore} disabled={loading}
        className="inline-flex items-center gap-2 rounded-[var(--radius-pill)] border border-ink px-5 py-2.5 text-[0.6875rem] font-medium uppercase tracking-[0.12em] text-ink enabled:hover:bg-ink enabled:hover:text-ground disabled:cursor-not-allowed disabled:opacity-60">
        {loading ? <T k="pages.shop.loading_more" /> : <><T k="pages.shop.more" /><Icon name="expand" size={14} /></>}
      </button>
    </div>
  );
}

export default function ShopResults({ items, total, hasMore, loading, error, lines, onMore, onRetry, onClear }: ShopResultsProps) {
  if (items.length === 0) {
    if (error) return <Failed onRetry={onRetry} />;
    if (loading) {
      return (
        <ShopNotice>
          <p role="status" className="max-w-[44ch] text-sm text-ink-soft"><T k="pages.shop.loading" /></p>
        </ShopNotice>
      );
    }
    if (total === 0) return <Empty onClear={onClear} />;
  }
  return (
    <>
      <Grid count={items.length} max={3}>
        {items.map((p) => <ProductCard key={p.id} product={p} line={lines.get(p.line) ?? null} />)}
      </Grid>
      {error ? <Failed onRetry={onRetry} className="mt-12" /> : hasMore && <ShowMore loading={loading} onMore={onMore} />}
    </>
  );
}
