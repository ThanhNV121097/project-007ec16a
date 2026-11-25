import { Link } from "react-router-dom";
import { T, useContent } from "../editable";
import { usePageMeta } from "../seo";
import { useCollection } from "../data";
import { formatPrice } from "../lib/format";
import { fill, sizesInStock, type Category, type Line, type Product } from "../lib/shop";
import { SplitHeading } from "../motion";
import ColourDots from "../components/ColourDots";
import Grid from "../components/Grid";
import Icon from "../components/Icon";
import OutlinePill from "../components/OutlinePill";
import ProductCard from "../components/ProductCard";
import ProductImage from "../components/ProductImage";
import PromiseRow from "../components/PromiseRow";
import Rule from "../components/Rule";

/**
 * THE HOME PAGE: one pair at full size — the template's own first screen,
 * its name large at the left and its tile filling the right half — then the
 * pairs new this month, who they are for, the lines as an index, and what
 * the shop promises. The pair that opens the site is the record marked
 * `featured` in src/data, chosen by the owner in the Data panel.
 */
export default function Home() {
  usePageMeta("pages.home.title", "pages.home.description");
  const all = useCollection<Product>("products", { where: { status: "live" }, pageSize: 500 }).items;
  const cats = useCollection<Category>("categories", { pageSize: 12 }).items;
  const lines = useCollection<Line>("lines", { pageSize: 24 }).items;
  const inStockText = useContent("pages.home.in_stock");
  const colourText = useContent("pages.home.colour");
  const pairsText = useContent("pages.home.pairs");
  const sinceText = useContent("pages.home.since");
  const giftImage = useContent("pages.home.gift_image");
  const giftName = useContent("pages.home.gift_name");
  const giftLine = useContent("pages.home.gift_line");
  const giftLimit = 3000000;

  const lead = all.find((p) => p.featured) ?? all[0];
  const fresh = all.filter((p) => p.is_new && p.id !== lead?.id).slice(0, 4);
  const siblings = lead ? all.filter((p) => p.model && p.model === lead.model) : [];
  const range = lead?.sizes?.length ? `EU ${lead.sizes[0]} – ${lead.sizes[lead.sizes.length - 1]}` : "";
  const gifts = all.filter((p) => p.price < giftLimit).length;

  return (
    <>
      {lead && (
        <section data-record={`products:${lead.id}`} className="grid gap-10 pb-12 pt-8 lg:grid-cols-12 lg:gap-8 lg:pb-14 lg:pt-10">
          <div className="flex flex-col lg:col-span-5 lg:pt-2">
            <T k="pages.home.eyebrow" as="p" className="text-[0.6875rem] font-medium uppercase tracking-[0.14em] text-ink-soft" />
            <SplitHeading k="pages.home.headline" as="h1" className="mt-4 max-w-[11ch] text-[length:var(--size-h1)] leading-[1.02] text-ink" />
            <p className="mt-5 text-[0.6875rem] font-medium uppercase tracking-[0.12em] text-ink-soft">{range}</p>
            <p className="mt-2 text-2xl tabular-nums text-ink">{formatPrice(lead.price)}</p>
            <div className="mt-3">
              <ColourDots label={`${colourText} — ${lead.colourway ?? ""}`} current={lead.id}
                items={siblings.map((s) => ({ id: s.id, name: s.name, swatch: s.swatch }))} />
            </div>
            <p className="mt-4 max-w-[38ch] text-sm leading-relaxed text-ink-soft">{lead.description}</p>
            <div className="mt-8 flex flex-wrap items-center gap-x-5 gap-y-3">
              <div data-magnetic="">
                <OutlinePill to={`/shop/${lead.id}`} testid="home-lead"><T k="pages.home.lead_cta" /></OutlinePill>
              </div>
              <p className="flex items-center gap-2 text-xs text-ink-soft">
                <Icon name="check" size={16} className="shrink-0 text-ink" />
                {fill(inStockText, { n: sizesInStock(lead).length })}
              </p>
            </div>
          </div>
          <div className="lg:col-span-6 lg:col-start-7">
            <Link to={`/shop/${lead.id}`} aria-label={lead.name} className="block overflow-hidden bg-surface">
              <div data-parallax="">
                <ProductImage src={lead.image} hover={lead.image_back} alt={lead.name} aspect="1/1" eager />
              </div>
            </Link>
          </div>
        </section>
      )}

      <Rule />
      <div className="flex items-center justify-between gap-4 py-7">
        <T k="pages.home.new_heading" as="h2" className="text-lg text-ink" />
        <OutlinePill to="/shop" testid="home-more"><T k="pages.home.new_cta" /></OutlinePill>
      </div>
      <Rule />
      <section className="py-10">
        <Grid count={fresh.length} max={4}>
          {fresh.map((p) => <ProductCard key={p.id} product={p} tilt />)}
        </Grid>
      </section>

      <Rule />
      <div className="flex items-center justify-between gap-4 py-7">
        <T k="pages.home.for_heading" as="h2" className="text-lg text-ink" />
      </div>
      <Rule />
      <section className="py-10">
        <Grid count={cats.length + 1} max={4}>
          {cats.map((c) => (
            <Link key={c.id} to={`/shop?category=${c.id}`} data-record={`categories:${c.id}`} className="group block">
              <ProductImage src={c.image} alt={c.name} aspect="1/1" />
              <span className="mt-3 flex items-baseline justify-between gap-3 text-[0.6875rem] font-medium uppercase tracking-[0.1em] text-ink">
                <span className="underline-offset-4 group-hover:underline">{c.name}</span>
                <span className="shrink-0 tabular-nums">{fill(pairsText, { n: all.filter((p) => p.category === c.id).length })}</span>
              </span>
              <span className="mt-1 block text-xs text-ink-soft">{c.blurb}</span>
            </Link>
          ))}
          <Link to="/shop?price=under-3m" className="group block">
            <ProductImage src={giftImage} alt={giftName} aspect="1/1" />
            <span className="mt-3 flex items-baseline justify-between gap-3 text-[0.6875rem] font-medium uppercase tracking-[0.1em] text-ink">
              <span className="underline-offset-4 group-hover:underline">{giftName}</span>
              <span className="shrink-0 tabular-nums">{fill(pairsText, { n: gifts })}</span>
            </span>
            <span className="mt-1 block text-xs text-ink-soft">{fill(giftLine, { price: formatPrice(giftLimit) })}</span>
          </Link>
        </Grid>
      </section>

      <Rule />
      <section className="grid gap-8 py-14 lg:grid-cols-12">
        <div className="lg:col-span-4">
          <T k="pages.home.lines_heading" as="h2" className="text-[length:var(--size-h2)] leading-[1.1] text-ink" />
          <T k="pages.home.lines_line" as="p" className="mt-3 max-w-[34ch] text-sm leading-relaxed text-ink-soft" />
        </div>
        <ul className="border-t border-line lg:col-span-8">
          {lines.map((l) => (
            <li key={l.id} data-record={`lines:${l.id}`} className="border-b border-line">
              <Link to={`/shop?line=${l.id}`} className="group flex items-baseline gap-4 py-5">
                <span className="min-w-0 flex-1 text-2xl tracking-tight text-ink underline-offset-8 group-hover:underline sm:text-3xl">{l.name}</span>
                <span className="hidden w-24 text-xs tabular-nums text-ink-soft sm:block">{l.since ? fill(sinceText, { year: l.since }) : ""}</span>
                <span className="w-20 text-right text-xs tabular-nums text-ink-soft">{fill(pairsText, { n: all.filter((p) => p.line === l.id).length })}</span>
                <Icon name="arrow" size={18} className="shrink-0 self-center text-ink transition-transform group-hover:translate-x-1" />
              </Link>
            </li>
          ))}
        </ul>
      </section>

      <section className="pb-6">
        <PromiseRow />
      </section>
    </>
  );
}
