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

  return (
    <>
      <OrderHead order={order} placed={params.has("placed")} />
      <Rule />
      <div className="grid gap-12 py-10 lg:grid-cols-12 lg:gap-8">
<<<<<<< refs/heads/feature/order-page-and-pay-now-ui
        <OrderLines order={order} />
        <OrderDetails order={order}
          method={payments.items.find((m) => m.id === order.payment)}
          option={deliveries.items.find((d) => d.id === order.delivery)} />
=======
        <div className="lg:col-span-7">
          <T k="pages.order.items" as="h2" className="text-lg text-ink" />
          <ul className="mt-5 border-t border-line">
            {view.lines.map((l) => (
              <li key={`${l.id}-${l.variant}`} data-record={`products:${l.id}`} className="flex gap-5 border-b border-line py-5">
                <div className="flex min-w-0 flex-1 items-start justify-between gap-4">
                  <div className="min-w-0">
                    <Link to={`/shop/${l.id}`} className="text-[0.6875rem] font-medium uppercase tracking-[0.1em] text-ink underline-offset-4 hover:underline">{l.name}</Link>
                    <p className="mt-1 text-xs text-ink-soft">{fill(lineText, { variant: l.variant ?? "", qty: l.qty })}</p>
                  </div>
                  <p className="shrink-0 text-sm tabular-nums text-ink">{formatPrice(l.price * l.qty)}</p>
                </div>
              </li>
            ))}
          </ul>
          <div className="flex items-baseline justify-between py-4">
            <T k="cart.total" className="text-sm text-ink" />
            <span className="text-2xl tabular-nums text-ink">{formatPrice(view.total)}</span>
          </div>
        </div>
        <aside className="flex flex-col gap-8 lg:col-span-5 lg:pl-8">
          <div>
            <T k="pages.order.details" as="h2" className="text-lg text-ink" />
            <SpecList className="mt-4" items={[
              [spec?.to, [view.customer, view.city].filter(Boolean).join(", ")],
              [spec?.delivery, option ? `${option.name} · ${option.days} · ${freeText}` : view.delivery],
              [spec?.payment, method ? `${method.name} · ${view.paid ? paidText : dueText}` : view.payment],
              [spec?.carrier, view.carrier],
              [spec?.tracking, view.tracking],
            ]} />
          </div>
          <Disclosure rows={[
            { label: <T k="pages.order.change" />, content: (
              <p className="max-w-[48ch] leading-relaxed"><T k="pages.order.change_text" /> <Link to="/contact" className="text-ink underline underline-offset-4"><T k="pages.order.change_link" /></Link></p>
            ) },
            { label: <T k="pages.order.wrong" />, content: (
              <p className="max-w-[48ch] leading-relaxed"><T k="pages.order.wrong_text" /> <Link to="/returns" className="text-ink underline underline-offset-4"><T k="pages.order.wrong_link" /></Link></p>
            ) },
          ]} />
          <div><OutlinePill to="/shop"><T k="pages.cart.keep" /></OutlinePill></div>
        </aside>
>>>>>>> refs/heads/main
      </div>
    </>
  );
}
