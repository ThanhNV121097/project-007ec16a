import { T } from "../editable";
import { usePageMeta } from "../seo";
import { useCollection } from "../data";
import { useTableQuery } from "../lib/table";
import type { Kind, Product } from "../lib/shop";
import FilterRail from "../components/FilterRail";
import Grid from "../components/Grid";
import ProductCard from "../components/ProductCard";
import Rule from "../components/Rule";

/**
 * The catalogue: how many there are, what to narrow them by, and the tiles.
 *
 * The filter lives in the address bar (useTableQuery), so "the shirts" is a
 * link somebody can send, and opening a product and coming back lands on the
 * same list. A filter that matches nothing says so and offers the way back —
 * never an empty page that looks like an empty shop.
 */
export default function Shop() {
  usePageMeta("pages.shop.title", "pages.shop.description");
  const t = useTableQuery({ size: 12 });
  const kinds = useCollection<Kind>("categories", { pageSize: 50 }).items;
  const { items, total, page, pages } = useCollection<Product>("products", {
    where: { category: t.list("category") }, page: t.page, pageSize: t.size,
  });
  return (
    <>
      <div className="flex items-end justify-between gap-6 py-10">
        <T k="pages.shop.heading" as="h1" className="text-[length:var(--size-h1)] leading-[1.02] text-ink" />
        <p data-testid="shop-count" className="shrink-0 pb-2 text-sm text-ink-soft">
          <span className="tabular-nums text-ink">{total}</span>{" "}<T k="pages.shop.count" />
        </p>
      </div>
      <Rule />
      <div className="flex flex-col gap-10 py-10 lg:flex-row lg:gap-16">
        <FilterRail
          className="lg:w-48 lg:shrink-0"
          clear={<T k="pages.shop.clear" />}
          groups={[{ key: "category", label: <T k="pages.shop.filter_kind" />, options: kinds.map((k) => [k.id, k.name] as const) }]}
        />
        <div className="min-w-0 flex-1">
          {items.length === 0 ? (
            <div className="py-16 text-center">
              <T k="pages.shop.empty" as="p" className="text-ink-soft" />
              <button type="button" onClick={t.clear} className="mt-4 text-sm text-ink underline underline-offset-8">
                <T k="pages.shop.clear" />
              </button>
            </div>
          ) : (
            <Grid max={3}>
              {items.map((p) => <ProductCard key={p.id} product={p} />)}
            </Grid>
          )}
          {pages > 1 && (
            <nav aria-label="Pages" className="mt-12 flex items-center justify-center gap-6 text-sm">
              <button type="button" disabled={page <= 1} onClick={() => t.setPage(page - 1)}
                className="text-ink underline-offset-8 hover:underline disabled:text-ink-soft disabled:no-underline">
                <T k="pages.shop.previous" />
              </button>
              <span className="tabular-nums text-ink-soft">{page} / {pages}</span>
              <button type="button" disabled={page >= pages} onClick={() => t.setPage(page + 1)}
                className="text-ink underline-offset-8 hover:underline disabled:text-ink-soft disabled:no-underline">
                <T k="pages.shop.next" />
              </button>
            </nav>
          )}
        </div>
      </div>
    </>
  );
}
