import { Link } from "react-router-dom";
import { T } from "../editable";
import { formatDate } from "../lib/format";
import type { StoredOrder } from "../lib/orders";

/**
 * "ORDERED ON THIS PHONE": the orders this browser holds a token for, each its
 * number and the date it was placed, linking to its own page. Nothing at all
 * is drawn when there are none: no heading, no empty box.
 */
export default function HeldOrders({ orders }: { orders: StoredOrder[] }) {
  if (orders.length === 0) return null;
  return (
    <div className="mt-10">
      <T k="pages.track.yours" as="h2" className="mb-3 text-lg text-ink" />
      <ul data-testid="track-orders" className="border-t border-line">
        {orders.map((o) => (
          <li key={o.id} className="border-b border-line">
            <Link to={`/orders/${o.id}`} className="flex items-center justify-between gap-4 py-3 text-sm text-ink hover:underline">
              <span className="tabular-nums">{o.number}</span>
              <span className="text-xs tabular-nums text-ink-soft">{formatDate(o.placed_at)}</span>
            </Link>
          </li>
        ))}
      </ul>
    </div>
  );
}
