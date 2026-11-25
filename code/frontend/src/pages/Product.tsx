import { useState } from "react";
import { Link, useParams } from "react-router-dom";
import { T, useContent } from "../editable";
import { usePageMeta } from "../seo";
import { useCollection, useRecord } from "../data";
import { addToCart } from "../lib/cart";
import { openBag } from "../lib/bag";
import { formatDate, formatPrice } from "../lib/format";
import { fill, picturesOf, sizesInStock, type Category, type Line, type PaymentMethod, type Product as Item, type Review, type ShopTerms } from "../lib/shop";
import Bleed from "../components/Bleed";
import ColourDots from "../components/ColourDots";
import Disclosure from "../components/Disclosure";
import Grid from "../components/Grid";
import Icon, { type IconName } from "../components/Icon";
import OutlinePill from "../components/OutlinePill";
import ProductCard from "../components/ProductCard";
import ProductGallery from "../components/ProductGallery";
import Rating from "../components/Rating";
import Rule from "../components/Rule";
import SizePicker from "../components/SizePicker";
import SpecList from "../components/SpecList";

type Words = Record<string, string>;

function useWords() {
  return {
    colour: useContent("pages.product.colour"),
    stockIn: useContent("pages.product.stock_in"),
    stockLow: useContent("pages.product.stock_low"),
    stockAll: useContent("pages.product.stock_all"),
    stockSome: useContent("pages.product.stock_some"),
    choose: useContent("pages.product.choose"),
    delivery: useContent("pages.product.delivery_text"),
    count: useContent("pages.product.review_count"),
    bought: useContent("pages.product.bought"),
    fit: useContent<Words>("pages.product.fit") ?? {},
    fitMost: useContent<Words>("pages.product.fit_most") ?? {},
    spec: useContent<Words>("pages.product.spec") ?? {},
  };
}
type ProductWords = ReturnType<typeof useWords>;

/** What the stock line under the sizes says, for the size chosen or for the pair as a whole. */
function stockLine(item: Item, size: string, warn: boolean, cutoff: string, w: ProductWords): { icon: IconName; text: string; alert: boolean } {
  const low = (item.sizes_low ?? []).filter((s) => !(item.sizes_out ?? []).includes(s));
  const n = sizesInStock(item).length;
  if (warn && !size) return { icon: "alert", text: w.choose, alert: true };
  if (size && low.includes(size)) return { icon: "alert", text: fill(w.stockLow, { size }), alert: true };
  if (size) return { icon: "check", text: fill(w.stockIn, { size, cutoff }), alert: false };
  if (low.length > 0) return { icon: "check", text: fill(w.stockSome, { n, low: low.join(", ") }), alert: false };
  return { icon: "check", text: fill(w.stockAll, { n }), alert: false };
}

/** The template's left column: name, price, colourways, sizes with stock, and the button. */
function BuyBox({ item, meta, siblings, cutoff, w }: { item: Item; meta: string; siblings: Item[]; cutoff: string; w: ProductWords }) {
  const [size, setSize] = useState("");
  const [warn, setWarn] = useState(false);
  const stock = stockLine(item, size, warn, cutoff, w);
  const add = () => {
    if (!size) {
      setWarn(true);
      return;
    }
    addToCart({ id: item.id, name: item.name, price: item.price, image: item.image, variant: `EU ${size}` });
    openBag();
  };
  return (
    <>
      <div>
        <h1 data-testid="product-name" className="max-w-[12ch] text-[length:var(--size-h1)] leading-[1.02] text-ink">{item.name}</h1>
        <p className="mt-4 text-[0.6875rem] font-medium uppercase tracking-[0.12em] text-ink-soft">{meta}</p>
        <div className="mt-3 flex flex-wrap items-baseline gap-x-4 gap-y-1">
          <p className="text-2xl tabular-nums text-ink">{formatPrice(item.price)}</p>
          {item.rating ? (
            <a href="#reviews" className="flex items-center gap-2 text-xs text-ink-soft hover:text-ink">
              <Rating value={item.rating} size={13} /> <span className="tabular-nums">{item.rating} · {fill(w.count, { n: item.review_count ?? 0 })}</span>
            </a>
          ) : null}
        </div>
      </div>
      <ColourDots label={`${w.colour} — ${item.colourway ?? ""}`} current={item.id}
        items={siblings.map((s) => ({ id: s.id, name: s.name, swatch: s.swatch }))} />
      <SizePicker sizes={item.sizes ?? []} low={item.sizes_low} out={item.sizes_out} value={size}
        onChange={(s) => { setSize(s); setWarn(false); }}
        label={<T k="pages.product.size" />} chart={<T k="pages.product.size_chart" />} />
      <p role="status" className={`-mt-2 flex items-start gap-2 text-xs ${stock.alert ? "text-danger" : "text-ink-soft"}`}>
        <Icon name={stock.icon} size={16} className={`shrink-0 ${stock.alert ? "text-danger" : "text-ink"}`} />{stock.text}
      </p>
      <div className="flex flex-col gap-3 sm:flex-row">
        <button type="button" onClick={add} data-testid="add-to-bag" data-magnetic=""
          className="inline-flex flex-1 items-center justify-center gap-2 whitespace-nowrap rounded-[var(--radius-pill)] bg-ink px-6 py-3.5 text-xs font-medium uppercase tracking-[0.14em] text-ground transition-opacity hover:opacity-85">
          <Icon name="bag" size={16} /><T k="pages.product.add" />
        </button>
        <OutlinePill to={`/contact?about=${encodeURIComponent(item.name)}`} testid="product-ask" className="justify-center py-3.5">
          <T k="pages.product.ask" />
        </OutlinePill>
      </div>
      {item.description && <p className="max-w-[44ch] text-sm leading-relaxed text-ink-soft">{item.description}</p>}
    </>
  );
}

/** The three disclosure rows: details, payment options, delivery and returns. */
function DetailRows({ item, line, cat, terms, payments, w }: {
  item: Item; line?: Line; cat?: Category; terms?: ShopTerms; payments: PaymentMethod[]; w: ProductWords;
}) {
  const s = w.spec;
  const values = { days: terms?.standard_days ?? "", cutoff: terms?.dispatch_cutoff ?? "" };
  return (
    <Disclosure rows={[
      { label: <T k="pages.product.details" />, content: (
        <SpecList items={[
          [s.line, line?.name], [s.for, cat?.name], [s.code, item.code], [s.colourway, item.colourway],
          [s.upper, item.upper], [s.lining, item.lining], [s.sole, item.sole], [s.fit, item.fit], [s.care, item.care],
        ]} />
      ) },
      { label: <T k="pages.product.payments" />, content: (
        <ul className="flex flex-col gap-3">
          {payments.map((m) => (
            <li key={m.id} className="flex gap-3">
              <Icon name={(m.icon ?? "card") as IconName} size={18} className="mt-0.5 shrink-0 text-ink" />
              <span><span className="text-ink">{m.name}.</span> {m.note}</span>
            </li>
          ))}
        </ul>
      ) },
      { label: <T k="pages.product.delivery" />, content: (
        <div className="flex flex-col gap-3">
          <p className="max-w-[52ch] leading-relaxed">{fill(w.delivery, values)}</p>
          <p className="flex gap-5">
            <Link to="/shipping" className="text-ink underline underline-offset-4"><T k="pages.product.delivery_link" /></Link>
            <Link to="/returns" className="text-ink underline underline-offset-4"><T k="pages.product.returns_link" /></Link>
          </p>
        </div>
      ) },
    ]} />
  );
}

/** What buyers said: the rating large at the left, three reviews on hairlines at the right. */
function Reviews({ item, reviews, w }: { item: Item; reviews: Review[]; w: ProductWords }) {
  const fits = reviews.map((r) => r.fit).filter((f): f is NonNullable<Review["fit"]> => Boolean(f));
  const mostFit = [...fits].sort((a, b) => fits.filter((f) => f === b).length - fits.filter((f) => f === a).length)[0];
  return (
    <section id="reviews" className="flex flex-col gap-10 py-12 lg:flex-row lg:gap-12">
      <div className="lg:w-1/3">
        <T k="pages.product.reviews" as="h2" className="text-lg text-ink" />
        <p className="mt-6 text-[length:var(--size-h1)] leading-none tabular-nums text-ink">{item.rating}</p>
        <Rating value={item.rating ?? 0} className="mt-4" />
        <p className="mt-3 text-sm text-ink-soft">{fill(w.count, { n: item.review_count ?? reviews.length })}</p>
        {mostFit && <p className="mt-1 text-sm text-ink-soft">{w.fitMost[mostFit]}</p>}
      </div>
      <ul className="border-t border-line lg:flex-1">
        {reviews.map((r) => (
          <li key={r.id} data-record={`reviews:${r.id}`} className="flex flex-col gap-3 border-b border-line py-6 sm:flex-row sm:gap-8">
            <div className="sm:w-36 sm:shrink-0">
              <p className="text-sm text-ink">{r.name}</p>
              <p className="text-xs tabular-nums text-ink-soft">{formatDate(r.date)}</p>
            </div>
            <div className="min-w-0 flex-1">
              <Rating value={r.rating} size={14} />
              <p className="mt-2 max-w-[60ch] text-sm leading-relaxed text-ink">{r.body}</p>
              <p className="mt-2 text-xs text-ink-soft">{fill(w.bought, { size: r.size_bought ?? "", fit: r.fit ? w.fit[r.fit] : "" })}</p>
            </div>
          </li>
        ))}
      </ul>
    </section>
  );
}

/** "Looking for more?" — the template's own closing row: a heading, a More pill, four tiles. */
function More({ items }: { items: Item[] }) {
  if (items.length === 0) return null;
  return (
    <>
      <Rule />
      <div className="flex items-center justify-between gap-4 py-7">
        <T k="pages.product.related" as="h2" className="text-lg text-ink" />
        <OutlinePill to="/shop"><T k="pages.product.related_cta" /></OutlinePill>
      </div>
      <Rule />
      <section className="py-10">
        <Grid count={items.length} max={4}>
          {items.map((p) => <ProductCard key={p.id} product={p} />)}
        </Grid>
      </section>
    </>
  );
}

/**
 * ONE PAIR, AS THE TEMPLATE'S PICTURE SETS IT: the name large at the left
 * with its price, its colourways as dots, the sizes with their stock and the
 * button, and the photographs at the right — then three disclosure rows,
 * what buyers said, and four more pairs. Each part draws only what the
 * record gives it.
 */
export default function Product() {
  usePageMeta("pages.product.title", "pages.product.description");
  const { id } = useParams();
  const { item } = useRecord<Item>("products", id);
  const all = useCollection<Item>("products", { where: { status: "live" }, pageSize: 500 }).items;
  const line = useRecord<Line>("lines", item?.line).item;
  const cat = useRecord<Category>("categories", item?.category).item;
  const terms = useRecord<ShopTerms>("shop", "shop").item;
  const payments = useCollection<PaymentMethod>("payments", { pageSize: 10 }).items;
  const reviews = useCollection<Review>("reviews", { where: { product: id ?? "", status: "live" }, sort: "-date", pageSize: 3 }).items;
  const w = useWords();

  if (!item) {
    return (
      <section className="flex flex-col items-start gap-6 py-24">
        <T k="pages.product.missing" as="h1" className="text-[length:var(--size-h2)] text-ink" />
        <OutlinePill to="/shop"><T k="pages.product.back" /></OutlinePill>
      </section>
    );
  }

  const siblings = all.filter((p) => p.model && p.model === item.model);
  const meta = [line?.name, cat?.name, item.code].filter(Boolean).join(" · ");
  const more = [
    ...all.filter((p) => p.id !== item.id && p.category === item.category),
    ...all.filter((p) => p.id !== item.id && p.category !== item.category),
  ].slice(0, 4);

  return (
    <>
      <section className="flex flex-col gap-10 pb-12 pt-6 lg:flex-row lg:gap-12 lg:pt-8">
        <div data-record={`products:${item.id}`} className="flex flex-col gap-6 lg:w-5/12">
          <Link to="/shop" className="inline-flex items-center gap-1.5 self-start text-[0.6875rem] font-medium uppercase tracking-[0.12em] text-ink-soft hover:text-ink">
            <Icon name="back" size={14} /><T k="pages.product.back" />
          </Link>
          <BuyBox key={item.id} item={item} meta={meta} siblings={siblings} cutoff={terms?.dispatch_cutoff ?? ""} w={w} />
          <DetailRows item={item} line={line} cat={cat} terms={terms} payments={payments} w={w} />
        </div>
        <div className="order-first lg:order-none lg:w-7/12">
          <Bleed side="right">
            <ProductGallery key={item.id} images={picturesOf(item)} alt={item.name} aspect="4/3"
              className="[&>[role=group]]:px-[var(--bleed)] lg:[&>[role=group]]:px-0" />
          </Bleed>
        </div>
      </section>
      {reviews.length > 0 && (
        <>
          <Rule />
          <Reviews item={item} reviews={reviews} w={w} />
        </>
      )}
      <More items={more} />
    </>
  );
}
