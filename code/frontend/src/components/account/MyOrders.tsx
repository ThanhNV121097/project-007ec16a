import { Link } from "react-router-dom";
import { T } from "../../editable";
import { STATUS_WORDS, W } from "../../lib/account-words";
import { formatDate, formatPrice } from "../../lib/format";
import type { OrderRow } from "../../lib/customer-accounts-sign-up-sign-in-my-orders";
import OutlinePill from "../OutlinePill";
import AccountBlock from "./AccountBlock";
import { ErrorLine } from "./Field";
import { FOCUS, OUTLINE } from "./styles";
import { type MyOrders as Orders } from "./useMyOrders";

/** One order as a hairline row: number and date at the left, total and status at the right; it opens the Order page. */
function Row({ order }: { order: OrderRow }) {
  return (
    <li className="border-b border-line">
      <Link to={`/orders/${order.id}`} data-testid="order-row" className={`flex min-h-11 items-start justify-between gap-4 py-4 hover:bg-wash ${FOCUS}`}>
        <span className="min-w-0">
          <span className="block text-sm tabular-nums text-ink">{order.number}</span>
          <span className="mt-1 block text-xs tabular-nums text-ink-soft">{formatDate(order.placed_at)}</span>
        </span>
        <span className="shrink-0 text-right">
          <span className="block text-sm tabular-nums text-ink">{formatPrice(order.total)}</span>
          <span className="mt-1 block text-xs text-ink-soft">{STATUS_WORDS[order.status] ?? order.status}</span>
        </span>
      </Link>
    </li>
  );
}

function Failed({ onRetry }: { onRetry: () => void }) {
  return (
    <div className="flex flex-col items-start gap-4">
      <ErrorLine>{W.orders.failed}</ErrorLine>
      <button type="button" onClick={onRetry} className={OUTLINE}>{W.orders.retry}</button>
    </div>
  );
}

/** MY ORDERS: loading, a list with "Show more", none yet, or could not load. */
export default function MyOrders({ orders }: { orders: Orders }) {
  const { phase, items, hasMore, loadingMore, moreFailed } = orders;
  return (
    <AccountBlock id="my-orders" title={W.orders.title}>
      {phase === "loading" && <p role="status" className="text-sm text-ink-soft">{W.orders.loading}</p>}
      {phase === "failed" && <Failed onRetry={orders.retry} />}
      {phase === "ready" && items.length === 0 && (
        <div className="flex flex-col items-start gap-5">
          <p className="text-sm text-ink-soft">{W.orders.empty}</p>
          <OutlinePill to="/shop" className="min-h-11"><T k="pages.cart.keep" /></OutlinePill>
        </div>
      )}
      {phase === "ready" && items.length > 0 && (
        <>
          <ul className="border-t border-line">{items.map((o) => <Row key={o.id} order={o} />)}</ul>
          {moreFailed && <div className="mt-6"><Failed onRetry={orders.showMore} /></div>}
          {hasMore && !moreFailed && (
            <button type="button" onClick={orders.showMore} disabled={loadingMore} className={`${OUTLINE} mt-6`}>{W.orders.more}</button>
          )}
        </>
      )}
    </AccountBlock>
  );
}
