import { useParams, useSearchParams } from "react-router-dom";
import { usePageMeta } from "../seo";
import { useCollection } from "../data";
import { useOrder } from "../lib/order-page-and-pay-now";
import type { DeliveryOption, PaymentMethod } from "../lib/shop";
import OrderDetails from "../components/OrderDetails";
import OrderHead from "../components/OrderHead";
import OrderLines from "../components/OrderLines";
import { OrderLoadError, OrderLoading, OrderMissing } from "../components/OrderStates";
import Rule from "../components/Rule";

/**
 * AFTER THE ORDER: the answer first and largest — placed, on its way,
 * delivered — then the five steps as one line, then the pairs, where they are
 * going and who to ask. The same page answers a buyer straight from checkout
 * and one who comes back to it.
 *
 * It draws only what the shop has said: nothing but a quiet block until the
 * answer for this order is in (and the names of its delivery and way to pay),
 * the not-found view after a 404 and only then, and one line with "Try again"
 * when the shop could not be asked.
 */
export default function Order() {
  usePageMeta("pages.order.title", "pages.order.description");
  const { id } = useParams();
  const [params] = useSearchParams();
  const read = useOrder(id);
  const payments = useCollection<PaymentMethod>("payments", { pageSize: 10 });
  const deliveries = useCollection<DeliveryOption>("delivery", { pageSize: 10 });

  if (read.state === "failed") return <OrderLoadError onRetry={read.reload} />;
  if (read.state === "missing") return <OrderMissing />;
  if (read.state === "loading" || payments.loading || deliveries.loading) return <OrderLoading />;
  const { order } = read;
  const method = payments.items.find((m) => m.id === order.payment);
  const option = deliveries.items.find((d) => d.id === order.delivery);

  return (
    <>
      <OrderHead order={order} placed={params.has("placed")} />
      <Rule />
      <div className="grid gap-12 py-10 lg:grid-cols-12 lg:gap-8">
        <OrderLines order={order} />
        <OrderDetails order={order} method={method} option={option} />
      </div>
    </>
  );
}
