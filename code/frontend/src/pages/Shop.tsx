import { useState, type FormEvent } from "react";
import { T, useContent } from "../editable";
import { usePageMeta } from "../seo";
import { useCollection } from "../data";
import { useTableQuery } from "../lib/table";
import { formatPrice } from "../lib/format";
import { fill, PRICE_BANDS, type Category, type Line } from "../lib/shop";
import { useProductFilters, useShopList } from "../lib/shop-list";
import { readShopQuery } from "../lib/shop-request";
import Bleed from "../components/Bleed";
import FilterRail from "../components/FilterRail";
import HeadPicture from "../components/HeadPicture";
import Icon from "../components/Icon";
import Select from "../components/Select";
import ShopFilterToggle from "../components/ShopFilterToggle";
import ShopResults from "../components/ShopResults";
import ShopSizes from "../components/ShopSizes";

/**
 * THE CATALOGUE: how many pairs there are, what to narrow them by, and the
 * tiles. Everything a visitor sets — who it is for, the line, the colour, a
 * price band, an EU size, the words searched and the order — lives in the
 * address bar, so "women's Samba in 38" is a link somebody can send and a
 * pair opened and closed returns to the same list, the first twelve of it.
 *
 * The list is read twelve live shoes at a time (src/lib/shop-list.ts); the
 * count is the answer's `total`, and the colours and sizes the rail offers
 * are every one a live shoe has, so they do not shrink as the list narrows.
 * What the results area shows while the list is on its way, when nothing
 * matches and when it could not be read is src/components/ShopResults.tsx.
 *
 * `size` here is an EU size, never a page size: the page opts in with
 * `sizeIsFilter`, so "Clear all" takes the pressed pills off with the rest.
 * Below 1024px the rail sits behind the "Filter" button.
 */
export default function Shop() {
  usePageMeta("pages.shop.title", "pages.shop.description");
  const t = useTableQuery({ sizeIsFilter: true });
  const query = readShopQuery(t);
  const cats = useCollection<Category>("categories", { pageSize: 20 }).items;
  // The header reads the lines with this page size too, so the two are one request.
  const lines = useCollection<Line>("lines", { pageSize: 20 }).items;
  const list = useShopList(query);
  const options = useProductFilters();
  const colourNames = useContent<Record<string, string>>("colours");
  const allTitle = useContent("pages.shop.heading");
  const headImage = useContent("pages.shop.head_image");
  const headAlt = useContent("pages.shop.head_alt");
  const resultsTitle = useContent("pages.shop.results");
  const searchLabel = useContent("pages.shop.search");
  const sortLabel = useContent("pages.shop.sort");
  const bandText = [useContent("pages.shop.band_under"), useContent("pages.shop.band_between"), useContent("pages.shop.band_over")];
  const sortText = {
    newest: useContent("pages.shop.sort_newest"),
    low: useContent("pages.shop.sort_low"),
    high: useContent("pages.shop.sort_high"),
    rated: useContent("pages.shop.sort_rated"),
  };
  const [filtersOpen, setFiltersOpen] = useState(false);

  const oneCat = query.category.length === 1 ? cats.find((c) => c.id === query.category[0]) : undefined;
  const oneLine = query.line.length === 1 ? lines.find((l) => l.id === query.line[0]) : undefined;
  const title = t.q ? fill(resultsTitle, { q: t.q }) : [oneCat?.name, oneLine?.name].filter(Boolean).join(" · ") || allTitle;
  // The number in "Filter (n)": every ticked box and every pressed size pill, one each. Search words and the order are not filters.
  const active = query.category.length + query.line.length + query.colour.length + query.price.length + query.size.length;
  const lineById = new Map(lines.map((l) => [l.id, l] as const));

  const bandLabel = (i: number) => {
    const b = PRICE_BANDS[i];
    return fill(bandText[i], { a: formatPrice(i === 0 ? b.to + 1 : b.from), b: formatPrice(b.to + 1) });
  };

  const search = (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    t.setQ(String(new FormData(e.currentTarget).get("q") ?? "").trim());
  };
  const toggleSize = (s: string) => t.set("size", query.size.includes(s) ? query.size.filter((x) => x !== s) : [...query.size, s]);

  return (
    <>
      <section className="flex flex-col gap-6 pb-8 pt-8 lg:flex-row lg:gap-8 lg:pt-10">
        <div className="flex flex-col justify-end gap-5 lg:shrink-0 lg:grow-0 lg:basis-1/2">
          <h1 className="min-w-0 text-[length:var(--size-h1)] leading-[1.02] text-ink">{title}</h1>
          {list.total === null ? (
            // No count for a list that has not come, and the heading does not jump when it does.
            <div aria-hidden="true" className="h-5" />
          ) : (
            <p data-testid="shop-count" className="text-sm text-ink-soft">
              <span data-count="" className="tabular-nums text-ink">{list.total}</span>{" "}<T k="pages.shop.count" />
            </p>
          )}
        </div>
        <HeadPicture src={oneCat?.image || headImage} alt={oneCat?.name || headAlt} className="lg:flex-1" />
      </section>
      <Bleed side="both" ground className="flex flex-col gap-3 py-4 sm:flex-row sm:items-center sm:justify-between">
        <form key={t.q} role="search" onSubmit={search} className="flex min-w-0 flex-1 items-center gap-3 sm:max-w-md">
          <Icon name="search" className="shrink-0 text-ink-soft" />
          <input id="shop-search" name="q" type="search" defaultValue={t.q} placeholder={searchLabel} aria-label={searchLabel}
            className="min-w-0 flex-1 appearance-none border-b border-field bg-transparent py-2 text-sm text-ink outline-none placeholder:text-ink-soft focus:border-ink" />
        </form>
        <div className="flex items-center gap-3">
          <ShopFilterToggle open={filtersOpen} count={active} controls="shop-filters" onToggle={() => setFiltersOpen((o) => !o)} />
          <Select aria-label={sortLabel} value={query.sort} onChange={(v) => t.set("sort", v === "-released" ? "" : v)}
            className="w-52"
            options={[
              { value: "-released", label: sortText.newest },
              { value: "price", label: sortText.low },
              { value: "-price", label: sortText.high },
              { value: "-rating", label: sortText.rated },
            ]} />
        </div>
      </Bleed>
      <div className="flex flex-col gap-10 py-8 lg:flex-row lg:gap-12">
        <div id="shop-filters" className={`${filtersOpen ? "block" : "hidden"} lg:block lg:w-56 lg:shrink-0`}>
          <FilterRail
            table={t}
            clear={<T k="pages.shop.clear" />}
            groups={[
              { key: "category", label: <T k="pages.shop.filter_for" />, options: cats.map((c) => [c.id, c.name] as const) },
              { key: "line", label: <T k="pages.shop.filter_line" />, options: lines.map((l) => [l.id, l.name] as const) },
              { key: "colour", label: <T k="pages.shop.filter_colour" />, options: options.colours.map((c) => [c, colourNames?.[c] ?? c] as const) },
              { key: "price", label: <T k="pages.shop.filter_price" />, options: PRICE_BANDS.map((b, i) => [b.id, bandLabel(i)] as const) },
            ]}
          />
          <ShopSizes sizes={options.sizes} pressed={query.size} onToggle={toggleSize} />
        </div>
        <div className="min-w-0 flex-1">
          <ShopResults items={list.items} total={list.total} hasMore={list.hasMore} loading={list.loading} error={list.error}
            lines={lineById} onMore={list.loadMore} onRetry={list.retry} onClear={t.clear} />
        </div>
      </div>
    </>
  );
}
