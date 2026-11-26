import { Link } from "react-router-dom";
import { T, useContent } from "../editable";
import { formatPrice } from "../lib/format";
import type { PublicOrder } from "../lib/order-page-and-pay-now";
import { fill } from "../lib/shop";

/**
 * "In this order": every pair with its name (a link to the shoe, even one
 * that is now a draft), its size and quantity, and its price as it was sold;
 * then the order's stored total — the shop's figure, not a sum made here.
 */
export default function OrderLines({ order }: { order: PublicOrder }) {
  const lineText = useContent("pages.checkout.line");
  return (
    <div className="min-w-0 lg:col-span-7">
      <T k="pages.order.items" as="h2" className="text-lg text-ink" />
      <ul className="mt-5 border-t border-line">
        {order.items.map((l, i) => (
          <li key={`${l.product}-${l.size}-${i}`} data-record={`products:${l.product}`} className="flex items-start justify-between gap-4 border-b border-line py-5">
            <div className="min-w-0">
              <Link to={`/shop/${l.product}`} className="wrap-anywhere text-[0.6875rem] font-medium uppercase tracking-[0.1em] text-ink underline-offset-4 hover:underline">{l.name}</Link>
              <p className="mt-1 text-xs text-ink-soft">{fill(lineText, { variant: `EU ${l.size}`, qty: l.qty })}</p>
            </div>
            <p className="shrink-0 text-sm tabular-nums text-ink">{formatPrice(l.unit_price * l.qty)}</p>
          </li>
        ))}
      </ul>
      <div className="flex items-baseline justify-between py-4">
        <T k="cart.total" className="text-sm text-ink" />
        <span data-testid="order-total" className="text-2xl tabular-nums text-ink">{formatPrice(order.total)}</span>
      </div>
    </div>
  );
}
