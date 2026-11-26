import { useContent } from "../editable";
import ProductImage from "./ProductImage";
import { formatDateTime, formatPrice } from "../lib/format";
import { fill } from "../lib/shop";
import type { DealerOrder } from "../lib/dealer-handles-orders-status-courier-payment";

/**
 * The top of an order's page: who, what, when, how much. One pair keeps the
 * design's sentence with the shoe's picture beside it; several pairs say how
 * many, and list each under the sentence — picture, name, EU size, quantity
 * and the price it was sold at. A row wraps rather than cuts on a phone.
 */
export default function OrderSummary({ order }: { order: DealerOrder }) {
  const one = useContent("pages.admin.orders.summary");
  const [first] = order.items;
  const many = order.items.length > 1;
  const date = formatDateTime(order.placed_at);
  const total = formatPrice(order.total);
  const sentence = many
    ? `${order.customer} in ${order.city} ordered ${order.items.length} pairs on ${date} — ${total}.`
    : fill(one, {
      customer: order.customer, city: order.city, qty: first?.qty ?? order.qty, product: first?.name ?? "",
      size: first?.size ?? order.size, date, total,
    });
  return (
    <div className="grid gap-4 rounded-md border border-line bg-surface p-4">
      <div className="flex items-center gap-4">
        {!many && (
          <span className="w-16 shrink-0"><ProductImage src={first?.image} alt={first?.name ?? ""} aspect="1/1" /></span>
        )}
        <p className="min-w-0 text-sm text-ink">{sentence}</p>
      </div>
      {many && (
        <ul className="grid gap-3">
          {order.items.map((p, i) => (
            <li key={`${p.product}-${p.size}-${i}`} className="flex flex-wrap items-center gap-x-4 gap-y-2 border-t border-line pt-3">
              <span className="w-12 shrink-0"><ProductImage src={p.image} alt="" aspect="1/1" /></span>
              <span className="min-w-0 flex-1 basis-40 text-sm">
                <span className="block font-medium text-ink">{p.name}</span>
                <span className="block text-xs text-ink-soft">{p.colourway}</span>
              </span>
              <span className="text-sm text-ink">{`EU ${p.size}`}</span>
              <span className="text-sm tabular-nums text-ink">{`× ${p.qty}`}</span>
              <span className="text-sm tabular-nums text-ink">{formatPrice(p.unit_price)}</span>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
