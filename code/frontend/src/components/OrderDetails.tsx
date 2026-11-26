import { Link } from "react-router-dom";
import { T, useContent } from "../editable";
import { canPay, type PublicOrder } from "../lib/order-page-and-pay-now";
import type { DeliveryOption, PaymentMethod } from "../lib/shop";
import Disclosure from "./Disclosure";
import OutlinePill from "./OutlinePill";
import PayNow from "./PayNow";
import SpecList from "./SpecList";

/** A value that may be long and unbroken (a customer's name): it wraps rather than widening the page. */
const wrapped = (text: string) => (text ? <span className="wrap-anywhere">{text}</span> : "");

/**
 * "Delivery and payment": where it goes, how it is sent and paid (by name —
 * a raw id is never drawn, so a row whose record is missing is left out), the
 * carrier and tracking number once they are set; then "Pay now" while the
 * order is unpaid and not cancelled, the two closed rows and the way back to
 * the shop. The address, phone, email and note are not here: the shop never
 * sends them.
 */
export default function OrderDetails({ order, method, option }: { order: PublicOrder; method?: PaymentMethod; option?: DeliveryOption }) {
  const spec = useContent<Record<string, string>>("pages.order.spec");
  const free = useContent("pages.cart.free");
  const paidText = useContent("pages.order.paid");
  const dueText = useContent("pages.order.due");
  return (
    <aside className="flex min-w-0 flex-col gap-8 lg:col-span-5 lg:pl-8">
      <div>
        <T k="pages.order.details" as="h2" className="text-lg text-ink" />
        <SpecList className="mt-4" items={[
          [spec?.to, wrapped([order.customer, order.city].filter(Boolean).join(", "))],
          [spec?.delivery, option ? `${option.name} · ${option.days} · ${free}` : ""],
          [spec?.payment, method ? `${method.name} · ${order.paid ? paidText : dueText}` : ""],
          [spec?.carrier, wrapped(order.carrier)],
          [spec?.tracking, wrapped(order.tracking)],
        ]} />
      </div>
      {canPay(order) && <PayNow orderId={order.id} />}
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
  );
}
