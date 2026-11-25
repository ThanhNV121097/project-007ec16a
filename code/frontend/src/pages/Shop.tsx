import { useState, type FormEvent } from "react";
import { T, useContent } from "../editable";
import { usePageMeta } from "../seo";
import { useCollection } from "../data";
import { useTableQuery } from "../lib/table";
import { formatPrice } from "../lib/format";
import { fill, inBand, PRICE_BANDS, sizeKey, sizesInStock, type Category, type Line, type Product } from "../lib/shop";
import Bleed from "../components/Bleed";
import FilterRail from "../components/FilterRail";
import Grid from "../components/Grid";
import HeadPicture from "../components/HeadPicture";
import Icon from "../components/Icon";
import ProductCard from "../components/ProductCard";
import Select from "../components/Select";
import { sizeColumns } from "../components/SizePicker";

/** "36 2/3" as a number, so sizes sort the way a foot grows. */
function sizeValue(s: string): number {
  const [whole, frac] = s.split(" ");
  if (!frac) return Number(whole);
  const [a, b] = frac.split("/").map(Number);
  return Number(whole) + (b ? a / b : 0);
}

/**
 * THE CATALOGUE: how many pairs there are, what to narrow them by, and the
 * tiles. Everything a visitor sets — who it is for, the line, the colour, a
 * price band, an EU size, the words searched and the order — lives in the
 * address bar, so "women's Samba in 38" is a link somebody can send and a
 * pair opened and closed returns to the same list.
 */
export default function Shop() {
  usePageMeta("pages.shop.title", "pages.shop.description");
  const t = useTableQuery({ size: 12 });
  const cats = useCollection<Category>("categories", { pageSize: 20 }).items;
  const lines = useCollection<Line>("lines", { pageSize: 30 }).items;
  const everything = useCollection<Product>("products", { where: { status: "live" }, pageSize: 500 }).items;
  const { items, loading } = useCollection<Product>("products", {
    where: { status: "live", category: t.list("category"), line: t.list("line"), colour: t.list("colour") },
    q: t.q,
    search: ["name", "model", "colourway", "code", "line", "category", "description"],
    sort: t.sort || "-released",
    pageSize: 500,
  });
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
  const [shown, setShown] = useState(12);
  const [filtersOpen, setFiltersOpen] = useState(false);

  const sizes = t.list("size");
  const bands = t.list("price");
  const matched = items.filter((p) =>
    (sizes.length === 0 || sizes.some((s) => sizesInStock(p).includes(s))) &&
    (bands.length === 0 || bands.some((b) => inBand(p.price, b))));
  const total = matched.length;

  const allSizes = [...new Set(everything.flatMap((p) => p.sizes ?? []))].sort((a, b) => sizeValue(a) - sizeValue(b));
  const colours = [...new Set(everything.map((p) => p.colour).filter((c): c is string => Boolean(c)))];
  const oneCat = t.list("category").length === 1 ? cats.find((c) => c.id === t.list("category")[0]) : undefined;
  const oneLine = t.list("line").length === 1 ? lines.find((l) => l.id === t.list("line")[0]) : undefined;
  const title = t.q ? fill(resultsTitle, { q: t.q }) : [oneCat?.name, oneLine?.name].filter(Boolean).join(" · ") || allTitle;
  const active = t.list("category").length + t.list("line").length + t.list("colour").length + bands.length + sizes.length;

  const bandLabel = (i: number) => {
    const b = PRICE_BANDS[i];
    return fill(bandText[i], { a: formatPrice(i === 0 ? b.to + 1 : b.from), b: formatPrice(b.to + 1) });
  };

  const search = (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    t.setQ(String(new FormData(e.currentTarget).get("q") ?? ""));
  };
  const toggleSize = (s: string) => t.set("size", sizes.includes(s) ? sizes.filter((x) => x !== s) : [...sizes, s]);

  return (
    <>
      <section className="flex flex-col gap-6 pb-8 pt-8 lg:flex-row lg:gap-8 lg:pt-10">
        <div className="flex flex-col justify-end gap-3 lg:shrink-0 lg:grow-0 lg:basis-1/2">
          <h1 className="min-w-0 text-[length:var(--size-h1)] leading-[1.02] text-ink">{title}</h1>
          <p data-testid="shop-count" className="text-sm text-ink-soft">
            <span data-count="" className="tabular-nums text-ink">{total}</span>{" "}<T k="pages.shop.count" />
          </p>
        </div>
        <HeadPicture src={oneCat?.image || headImage} alt={oneCat?.name || headAlt} className="lg:col-span-6 lg:col-start-7" />
      </section>
      <Bleed side="both" ground className="flex flex-col gap-3 py-4 sm:flex-row sm:items-center sm:justify-between">
        <form key={t.q} role="search" onSubmit={search} className="flex min-w-0 flex-1 items-center gap-3 sm:max-w-md">
          <Icon name="search" className="shrink-0 text-ink-soft" />
          <input id="shop-search" name="q" type="search" defaultValue={t.q} placeholder={searchLabel} aria-label={searchLabel}
            className="min-w-0 flex-1 appearance-none border-b border-field bg-transparent py-2 text-sm text-ink outline-none placeholder:text-ink-soft focus:border-ink" />
        </form>
        <div className="flex items-center gap-3">
          <button type="button" onClick={() => setFiltersOpen((o) => !o)} aria-expanded={filtersOpen}
            className="inline-flex items-center gap-2 rounded-[var(--radius-pill)] border border-ink px-4 py-2 text-[0.6875rem] font-medium uppercase tracking-[0.12em] text-ink lg:hidden">
            <Icon name="filter" size={14} /><T k="pages.shop.filters" />{active > 0 && <span className="tabular-nums">({active})</span>}
          </button>
          <Select aria-label={sortLabel} value={t.sort || "-released"} onChange={(v) => t.set("sort", v === "-released" ? "" : v)}
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
        <aside className={`${filtersOpen ? "block" : "hidden"} lg:block lg:w-56 lg:shrink-0`}>
          <FilterRail
            clear={<T k="pages.shop.clear" />}
            groups={[
              { key: "category", label: <T k="pages.shop.filter_for" />, options: cats.map((c) => [c.id, c.name] as const) },
              { key: "line", label: <T k="pages.shop.filter_line" />, options: lines.map((l) => [l.id, l.name] as const) },
              { key: "colour", label: <T k="pages.shop.filter_colour" />, options: colours.map((c) => [c, colourNames?.[c] ?? c] as const) },
              { key: "price", label: <T k="pages.shop.filter_price" />, options: PRICE_BANDS.map((b, i) => [b.id, bandLabel(i)] as const) },
            ]}
          />
          <div className="mt-6 border-b border-line pb-5">
            <T k="pages.shop.filter_size" as="p" className="pb-3 text-xs uppercase tracking-wider text-ink-soft" />
            <div className={`grid gap-1.5 ${sizeColumns(allSizes.length, 4)}`}>
              {allSizes.map((s) => {
                const on = sizes.includes(s);
                return (
                  <button key={s} type="button" aria-pressed={on} onClick={() => toggleSize(s)} data-testid={`filter-size-${sizeKey(s)}`}
                    className={`whitespace-nowrap rounded-[var(--radius-pill)] border px-1 py-1.5 text-[0.8125rem] tabular-nums ${on ? "border-ink bg-ink text-ground" : "border-line text-ink hover:border-ink"}`}>
                    {s}
                  </button>
                );
              })}
            </div>
          </div>
        </aside>
        <div className="min-w-0 flex-1">
          {!loading && total === 0 ? (
            <div className="flex flex-col items-start gap-5 border-b border-line pb-16 pt-6">
              <T k="pages.shop.empty" as="h2" className="text-[length:var(--size-h2)] text-ink" />
              <T k="pages.shop.empty_line" as="p" className="max-w-[44ch] text-sm text-ink-soft" />
              <button type="button" onClick={t.clear}
                className="inline-flex items-center gap-2 rounded-[var(--radius-pill)] border border-ink px-4 py-2 text-[0.6875rem] font-medium uppercase tracking-[0.12em] text-ink hover:bg-ink hover:text-ground">
                <T k="pages.shop.clear" /><Icon name="arrow" size={14} />
              </button>
            </div>
          ) : (
            <Grid count={Math.min(shown, total)} max={3}>
              {matched.slice(0, shown).map((p) => <ProductCard key={p.id} product={p} />)}
            </Grid>
          )}
          {total > shown && (
            <div className="mt-12 flex justify-center">
              <button type="button" onClick={() => setShown((n) => n + 12)}
                className="inline-flex items-center gap-2 rounded-[var(--radius-pill)] border border-ink px-5 py-2.5 text-[0.6875rem] font-medium uppercase tracking-[0.12em] text-ink hover:bg-ink hover:text-ground">
                <T k="pages.shop.more" /><Icon name="expand" size={14} />
              </button>
            </div>
          )}
        </div>
      </div>
    </>
  );
}
