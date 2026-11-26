import { Link } from "react-router-dom";
import { useContent } from "../editable";
import { formatDateTime, formatPrice } from "../lib/format";
import type { LatestOrder } from "../lib/overview";

/**
 * The latest orders, each a sentence a person can read at a glance: number,
 * customer, shoe and size at the left; status, time and total at the right,
 * the two halves wrapping onto two lines on a phone. Each row opens its order.
 */
export default function AdminRecentOrders({ rows }: { rows: readonly LatestOrder[] }) {
  const steps = useContent<Record<string, string>>("orders.steps");
  const cancelled = useContent("pages.admin.orders.cancelled");
  const statuses: Record<string, string> = { ...steps, cancelled };

  return (
    <ul className="divide-y divide-line">
      {rows.map((o) => (
        <li key={o.id}>
          <Link
            to={`/admin/orders/${o.id}`}
            className="flex min-h-11 flex-wrap content-center items-baseline justify-between gap-x-4 gap-y-1 py-2.5 text-sm hover:bg-wash"
          >
            <span className="min-w-0 break-words">
              <span className="font-medium tabular-nums text-ink">{o.number}</span>{" · "}
              <span className="text-ink">{o.customer}</span>{" · "}
              <span className="text-ink-soft">{`${o.product_name} · EU ${o.size}`}</span>
            </span>
            <span className="shrink-0 text-ink-soft">
              {statuses[o.status] ?? o.status} · <span className="tabular-nums">{formatDateTime(o.placed_at)}</span> · <span className="tabular-nums text-ink">{formatPrice(o.total)}</span>
            </span>
          </Link>
        </li>
      ))}
    </ul>
  );
}
