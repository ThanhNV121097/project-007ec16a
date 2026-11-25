import { useState } from "react";
import { Link, useParams } from "react-router-dom";
import { T, useContent } from "../editable";
import { usePageMeta } from "../seo";
import { useCollection, useRecord } from "../data";
import { addToCart } from "../lib/cart";
import { openBag } from "../lib/bag";
import { formatDate, formatPrice } from "../lib/format";
import { fill, picturesOf, sizesInStock, type Category, type Line, type PaymentMethod, type Product as Item, type Review, type ShopTerms } from "../lib/shop";
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
  const [size, setSize] = useState("");
  const [warn, setWarn] = useState(false);
  const words = {
    colour: useContent("pages.product.colour"),
    stockIn: useContent("pages.product.stock_in"),
    stockLow: useContent("pages.product.stock_low"),
    stockAll: useContent("pages.product.stock_all"),
    stockSome: useContent("pages.product.stock_some"),
    choose: useContent("pages.product.choose"),
    delivery: useContent("pages.product.delivery_text"),
    count: useContent("pages.product.review_count"),
    bought: useContent("pages.product.bought"),
    fit: useContent<Record<string, string>>("pages.product.fit"),
    spec: useContent<Record<string, string>>("pages.product.spec"),
  };

  if (!item) {
    return (
      <section className="flex flex-col items-start gap-6 py-24">
        <T k="pages.product.missing" as="h1" className="text-[length:var(--size-h2)] text-ink" />
        <OutlinePill to="/shop"><T k="pages.product.back" /></OutlinePill>
      </section>
    );
  }

  const siblings = all.filter((p) => p.model && p.model === item.model);
  const low = (item.sizes_low ?? []).filter((s) => !(item.sizes_out ?? []).includes(s));
  const inStock = sizesInStock(item);
  const cutoff = terms?.dispatch_cutoff ?? "";
  const stock: { icon: IconName; text: string; alert: boolean } = warn && !size
    ? { icon: "alert", text: words.choose, alert: true }
    : size
      ? low.includes(size)
        ? { icon: "alert", text: fill(words.stockLow, { size }), alert: true }
        : { icon: "check", text: fill(words.stockIn, { size, cutoff }), alert: false }
      : low.length > 0
        ? { icon: "check", text: fill(words.stockSome, { n: inStock.length, low: low.join(", ") }), alert: false }
        : { icon: "check", text: fill(words.stockAll, { n: inStock.length }), alert: false };

  const add = () => {
    if (!size) {
      setWarn(true);
      return;
    }
    addToCart({ id: item.id, name: item.name, price: item.price, image: item.image, variant: `EU ${size}` });
    openBag();
  };

  const more = [
    ...all.filter((p) => p.id !== item.id && p.category === item.category),
    ...all.filter((p) => p.id !== item.id && p.category !== item.category),
  ].slice(0, 4);
  const fits = reviews.map((r) => r.fit).filter(Boolean);
  const mostFit = fits.sort((a, b) => fits.filter((f) => f === b).length - fits.filter((f) => f === a).length)[0];

  return (
    <>
      <section className="grid gap-10 pb-12 pt-6 lg:grid-cols-12 lg:gap-8 lg:pt-8">
        <div data-record={`products:${item.id}`} className="flex flex-col gap-6 lg:col-span-5">
          <Link to="/shop" className="inline-flex items-center gap-1.5 self-start text-[0.6875rem] font-medium uppercase tracking-[0.12em] text-ink-soft hover:text-ink">
            <Icon name="back" size={14} /><T k="pages.product.back" />
          </Link>
          <div>
            <h1 data-testid="product-name" className="max-w-[12ch] text-[length:var(--size-h1)] leading-[1.02] text-ink">{item.name}</h1>
            <p className="mt-4 text-[0.6875rem] font-medium uppercase tracking-[0.12em] text-ink-soft">
              {[line?.name, cat?.name, item.code].filter(Boolean).join(" · ")}
            </p>
            <div className="mt-3 flex flex-wrap items-baseline gap-x-4 gap-y-1">
              <p className="text-2xl tabular-nums text-ink">{formatPrice(item.price)}</p>
              {item.rating ? (
                <a href="#reviews" className="flex items-center gap-2 text-xs text-ink-soft hover:text-ink">
                  <Rating value={item.rating} size={13} /> <span className="tabular-nums">{item.rating} · {fill(words.count, { n: item.review_count ?? 0 })}</span>
                </a>
              ) : null}
            </div>
          </div>
          <ColourDots label={`${words.colour} — ${item.colourway ?? ""}`} current={item.id}
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
          <Disclosure rows={[
            { label: <T k="pages.product.details" />, content: (
              <SpecList items={[
                [words.spec?.line, line?.name],
                [words.spec?.for, cat?.name],
                [words.spec?.code, item.code],
                [words.spec?.colourway, item.colourway],
                [words.spec?.upper, item.upper],
                [words.spec?.lining, item.lining],
                [words.spec?.sole, item.sole],
                [words.spec?.fit, item.fit],
                [words.spec?.care, item.care],
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
                <p className="max-w-[52ch] leading-relaxed">{fill(words.delivery, {
                  days: terms?.standard_days ?? "", cutoff, n: terms?.exchange_days ?? "", refund: terms?.refund_days ?? "",
                })}</p>
                <p className="flex gap-5">
                  <Link to="/shipping" className="text-ink underline underline-offset-4"><T k="pages.product.delivery_link" /></Link>
                  <Link to="/returns" className="text-ink underline underline-offset-4"><T k="pages.product.returns_link" /></Link>
                </p>
              </div>
            ) },
          ]} />
        </div>
        <div className="lg:col-span-7">
          <ProductGallery images={picturesOf(item)} alt={item.name} aspect="1/1" />
        </div>
      </section>

      {reviews.length > 0 && (
        <>
          <Rule />
          <section id="reviews" className="grid gap-10 py-12 lg:grid-cols-12">
            <div className="lg:col-span-4">
              <T k="pages.product.reviews" as="h2" className="text-lg text-ink" />
              <p className="mt-6 text-[length:var(--size-h1)] leading-none tabular-nums text-ink">{item.rating}</p>
              <Rating value={item.rating ?? 0} className="mt-4" />
              <p className="mt-3 text-sm text-ink-soft">{fill(words.count, { n: item.review_count ?? reviews.length })}</p>
              {mostFit && <p className="mt-1 text-sm text-ink-soft">{words.fit?.[mostFit]}</p>}
            </div>
            <ul className="border-t border-line lg:col-span-8">
              {reviews.map((r) => (
                <li key={r.id} data-record={`reviews:${r.id}`} className="grid gap-3 border-b border-line py-6 sm:grid-cols-4">
                  <div>
                    <p className="text-sm text-ink">{r.name}</p>
                    <p className="text-xs tabular-nums text-ink-soft">{formatDate(r.date, "medium")}</p>
                  </div>
                  <div className="sm:col-span-3">
                    <Rating value={r.rating} size={14} />
                    <p className="mt-2 max-w-[60ch] text-sm leading-relaxed text-ink">{r.body}</p>
                    <p className="mt-2 text-xs text-ink-soft">{fill(words.bought, { size: r.size_bought ?? "", fit: (r.fit && words.fit?.[r.fit]) || "" })}</p>
                  </div>
                </li>
              ))}
            </ul>
          </section>
        </>
      )}

      {more.length > 0 && (
        <>
          <Rule />
          <div className="flex items-center justify-between gap-4 py-7">
            <T k="pages.product.related" as="h2" className="text-lg text-ink" />
            <OutlinePill to="/shop"><T k="pages.product.related_cta" /></OutlinePill>
          </div>
          <Rule />
          <section className="py-10">
            <Grid count={more.length} max={4}>
              {more.map((p) => <ProductCard key={p.id} product={p} />)}
            </Grid>
          </section>
        </>
      )}
    </>
  );
}
