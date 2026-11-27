import type { ReactNode } from "react";
import { Badge } from "./admin/index";
import { formatDateTime, formatPrice } from "../lib/format";
import type { DealerOrder } from "../lib/dealer-handles-orders-status-courier-payment";

function Fact({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="grid content-start gap-1">
      <dt className="text-xs text-ink-soft">{label}</dt>
      <dd className="min-w-0 break-words text-sm text-ink">{children}</dd>
    </div>
  );
}

/**
 * What the shop alone decides about an order, as plain text: its number, when
 * it was placed, its pairs, its total, the way to pay and whether the payment
 * provider has confirmed it. No control here changes any of them — only
 * VNPay's own notice marks an order paid.
 */
export default function OrderFacts({ order, paymentName }: { order: DealerOrder; paymentName: string }) {
  return (
    <dl className="grid gap-x-4 gap-y-4 sm:grid-cols-2">
      <Fact label="Order number">{order.number}</Fact>
      <Fact label="Placed">{formatDateTime(order.placed_at)}</Fact>
      <Fact label="Shoe">
        <ul className="grid gap-1">
          {order.items.map((p, i) => (
            <li key={`${p.product}-${p.size}-${i}`}>{`${p.name}, EU ${p.size} × ${p.qty}`}</li>
          ))}
        </ul>
      </Fact>
      <Fact label="Total"><span className="tabular-nums">{formatPrice(order.total)}</span></Fact>
      <Fact label="Payment">{paymentName}</Fact>
      <Fact label="Payment status">
        <Badge tone={order.paid ? "good" : "warn"}>{order.paid ? "Paid" : "Awaiting payment"}</Badge>
        {order.transaction_no !== "" && <p className="mt-2">{`VNPay transaction ${order.transaction_no}`}</p>}
      </Fact>
    </dl>
  );
}
