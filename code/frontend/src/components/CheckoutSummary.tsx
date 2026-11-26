import { T, useContent } from "../editable";
import { lineKey, type CartLine } from "../lib/cart";
import { formatPrice } from "../lib/format";
import { fill } from "../lib/shop";
import Icon from "./Icon";
import ProductImage from "./ProductImage";

/**
 * "YOUR ORDER", KEPT IN VIEW BESIDE THE FORM: each pair at the shop's price
 * today, what it all comes to (delivery is free, so the total is the pairs'),
 * what happens after the button, whatever stopped the last press, and the
 * button. `problems` is every line that stopped it — the details to fill in,
 * the pairs the shop could not set aside, the order that did not go through —
 * one per line, announced as an alert.
 */
export default function CheckoutSummary({ lines, subtotal, method, problems, placing, blocked }: {
  lines: CartLine[];
  subtotal: number;
  /** The way to pay's name, once the shop has said what it offers. */
  method?: string;
  problems: string[];
  placing: boolean;
  /** The one way to pay is not available: the button is off. */
  blocked: boolean;
}) {
  const free = useContent("pages.cart.free");
  const size = useContent("pages.checkout.line");
  const next = useContent("pages.checkout.next_online");
  const paying = useContent("pages.checkout.paying");
  return (
    <aside className="lg:col-span-5 lg:pl-8">
      <T k="pages.checkout.summary" as="h2" className="text-lg text-ink" />
      <ul className="mt-5 border-t border-line">
        {lines.map((l) => (
          <li key={lineKey(l)} data-record={`products:${l.id}`} className="flex gap-4 border-b border-line py-4">
            <div className="w-20 shrink-0"><ProductImage src={l.image} alt={l.name} aspect="1/1" /></div>
            <div className="flex min-w-0 flex-1 items-start justify-between gap-3">
              <div className="min-w-0">
                <p className="text-[0.6875rem] font-medium uppercase tracking-[0.1em] text-ink">{l.name}</p>
                <p className="mt-1 text-sm text-ink-soft">{fill(size, { variant: l.variant ?? "", qty: l.qty })}</p>
              </div>
              <p className="shrink-0 text-sm tabular-nums text-ink">{formatPrice(l.price * l.qty)}</p>
            </div>
          </li>
        ))}
      </ul>
      <dl className="text-sm">
        <div className="flex justify-between border-b border-line py-3"><dt className="text-ink-soft"><T k="pages.cart.subtotal" /></dt><dd className="tabular-nums text-ink">{formatPrice(subtotal)}</dd></div>
        <div className="flex justify-between border-b border-line py-3"><dt className="text-ink-soft"><T k="pages.cart.delivery" /></dt><dd className="tabular-nums text-ink">{free}</dd></div>
        <div className="flex items-baseline justify-between py-4"><dt className="text-ink"><T k="cart.total" /></dt><dd data-testid="checkout-total" className="text-2xl tabular-nums text-ink">{formatPrice(subtotal)}</dd></div>
      </dl>
      <p className="flex items-start gap-2 border-t border-line pt-4 text-sm leading-relaxed text-ink-soft">
        <Icon name="info" size={16} className="shrink-0 text-ink" />
        {method !== undefined && fill(next, { method })}
      </p>
      {problems.length > 0 && (
        <ul role="alert" className="mt-4 flex flex-col gap-1 text-xs text-danger">
          {problems.map((p) => <li key={p} className="flex items-center gap-2"><Icon name="alert" size={14} className="shrink-0" />{p}</li>)}
        </ul>
      )}
      <button type="submit" data-testid="place-order" disabled={placing || blocked}
        className="mt-5 flex w-full items-center justify-center gap-2 rounded-[var(--radius-pill)] bg-ink px-6 py-3.5 text-xs font-medium uppercase tracking-[0.14em] text-ground hover:opacity-85 disabled:opacity-60">
        <Icon name="lock" size={16} />{placing ? paying : <T k="pages.checkout.place" />}
      </button>
      <T k="pages.checkout.terms" as="p" className="mt-3 text-sm leading-relaxed text-ink-soft" />
    </aside>
  );
}
