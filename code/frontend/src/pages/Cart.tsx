import { Link } from "react-router-dom";
import { T, useContent } from "../editable";
import { usePageMeta } from "../seo";
import { useCollection } from "../data";
import { lineKey, removeFromCart, setQuantity, useCart, type CartLine } from "../lib/cart";
import { useDemo } from "../lib/demo";
import { formatPrice } from "../lib/format";
import { fill, type Product } from "../lib/shop";
import HeadPicture from "../components/HeadPicture";
import EmptyBag from "../components/EmptyBag";
import Icon from "../components/Icon";
import OutlinePill from "../components/OutlinePill";
import ProductImage from "../components/ProductImage";
import PromiseRow from "../components/PromiseRow";
import QuantityStepper from "../components/QuantityStepper";
import Rule from "../components/Rule";

/**
 * THE BAG: every pair with its tile, its size and how many, and beside them
 * what it all comes to — delivery said before checkout, not after. An empty
 * bag is a page too: it says so and puts the new pairs in front of you.
 */
export default function Cart() {
  usePageMeta("pages.cart.title", "pages.cart.description");
  const { lines: real } = useCart();
  const demo = useDemo();
  const products = useCollection<Product>("products", { where: { status: "live" }, pageSize: 500 }).items;
  const countText = useContent("pages.cart.count");
  const freeText = useContent("pages.cart.free");
  const removeText = useContent("pages.cart.remove");
  const eachText = useContent("pages.cart.each");
  const headAlt = useContent("pages.cart.head_alt");

  const example: CartLine[] = products.slice(1, 3).map((p) => ({
    id: p.id, name: p.name, price: p.price, image: p.image, variant: `EU ${(p.sizes ?? [])[3] ?? ""}`, qty: 1,
  }));
  const showingExample = demo && real.length === 0;
  const lines = showingExample ? example : real;
  const count = lines.reduce((n, l) => n + l.qty, 0);
  const subtotal = lines.reduce((s, l) => s + l.price * l.qty, 0);

  if (lines.length === 0) return <EmptyBag />;

  return (
    <>
      <section className="grid gap-6 pb-8 pt-8 lg:grid-cols-12 lg:gap-8 lg:pt-10">
        <div className="flex flex-col justify-end gap-3 lg:col-span-6">
          <T k="pages.cart.heading" as="h1" className="text-[length:var(--size-h1)] leading-[1.02] text-ink" />
          <p className="text-sm tabular-nums text-ink-soft">{fill(countText, { n: count })}</p>
        </div>
        <HeadPicture src="/images/about-lacing.jpg" alt={headAlt} className="lg:col-span-6 lg:col-start-7" />
      </section>
      <Rule />
      <div className="grid gap-12 py-10 lg:grid-cols-12 lg:gap-8">
        <ul className="border-t border-line lg:col-span-7">
          {lines.map((l) => (
            <li key={lineKey(l)} data-record={`products:${l.id}`} className="flex gap-5 border-b border-line py-6">
              <Link to={`/shop/${l.id}`} className="w-28 shrink-0 sm:w-40" aria-label={l.name}>
                <ProductImage src={l.image} alt={l.name} aspect="1/1" />
              </Link>
              <div className="flex min-w-0 flex-1 flex-col">
                <div className="flex items-start justify-between gap-4">
                  <div className="min-w-0">
                    <Link to={`/shop/${l.id}`} className="text-[0.6875rem] font-medium uppercase tracking-[0.1em] text-ink underline-offset-4 hover:underline">{l.name}</Link>
                    <p className="mt-1 text-xs text-ink-soft">{l.variant}</p>
                    <p className="mt-1 text-xs tabular-nums text-ink-soft">{fill(eachText, { price: formatPrice(l.price) })}</p>
                  </div>
                  <p className="shrink-0 text-sm tabular-nums text-ink">{formatPrice(l.price * l.qty)}</p>
                </div>
                <div className="mt-auto flex items-center justify-between pt-5">
                  <QuantityStepper value={l.qty} max={5} onChange={(q) => { if (!showingExample) setQuantity(lineKey(l), q); }} />
                  <button type="button" onClick={() => { if (!showingExample) removeFromCart(lineKey(l)); }}
                    aria-label={`${removeText} ${l.name}`} className="inline-flex items-center gap-1.5 p-2 text-xs text-ink-soft hover:text-ink">
                    <Icon name="remove" size={16} /><span className="hidden sm:inline">{removeText}</span>
                  </button>
                </div>
              </div>
            </li>
          ))}
        </ul>
        <aside className="lg:col-span-5 lg:pl-8">
          <T k="pages.cart.summary" as="h2" className="text-lg text-ink" />
          <dl className="mt-5 border-t border-line text-sm">
            <div className="flex justify-between border-b border-line py-3"><dt className="text-ink-soft"><T k="pages.cart.subtotal" /></dt><dd className="tabular-nums text-ink">{formatPrice(subtotal)}</dd></div>
            <div className="flex justify-between border-b border-line py-3"><dt className="text-ink-soft"><T k="pages.cart.delivery" /></dt><dd className="tabular-nums text-ink">{freeText}</dd></div>
            <div className="flex items-baseline justify-between py-4"><dt className="text-ink"><T k="cart.total" /></dt><dd className="text-2xl tabular-nums text-ink">{formatPrice(subtotal)}</dd></div>
          </dl>
          <Link to="/checkout" data-testid="cart-checkout"
            className="mt-2 flex w-full items-center justify-center gap-2 rounded-[var(--radius-pill)] bg-ink px-6 py-3.5 text-xs font-medium uppercase tracking-[0.14em] text-ground hover:opacity-85">
            <Icon name="lock" size={16} /><T k="cart.pay" />
          </Link>
          <T k="pages.cart.pay_line" as="p" className="mt-3 text-xs leading-relaxed text-ink-soft" />
          <div className="mt-8"><PromiseRow compact /></div>
          <div className="mt-8"><OutlinePill to="/shop"><T k="pages.cart.keep" /></OutlinePill></div>
        </aside>
      </div>
    </>
  );
}
