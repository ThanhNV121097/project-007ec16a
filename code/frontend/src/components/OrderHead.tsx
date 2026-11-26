import { Link } from "react-router-dom";
import { useContent } from "../editable";
import { formatDate, formatDateTime } from "../lib/format";
import { vietnamTime, type PublicOrder } from "../lib/order-page-and-pay-now";
import { fill } from "../lib/shop";
import Bleed from "./Bleed";
import OrderSteps from "./OrderSteps";
import ProductImage from "./ProductImage";

/** The line under the answer: when it arrives, or that it has. Nothing for a cancelled order or one with no date. */
function etaLine(order: PublicOrder, eta: string, done: string): string {
  if (order.status === "delivered") return done;
  if (order.status === "cancelled" || !order.eta) return "";
  return fill(eta, { date: formatDate(order.eta) });
}

/**
 * The answer at the left — the number and when it was placed, the heading by
 * status (or the thanks, when the address carries `placed`), the line under
 * it and the five steps — and the first pair running off the right edge from
 * 1024px. A first pair with no picture draws no picture and no box, and the
 * two left-hand blocks both start at column 1 so they stack instead of the
 * steps moving up beside the heading into the space the picture left.
 */
export default function OrderHead({ order, placed }: { order: PublicOrder; placed: boolean }) {
  const headlines = useContent<Record<string, string>>("pages.order.headline");
  const thanks = useContent("pages.order.thanks");
  const numberText = useContent("pages.order.number");
  const etaText = useContent("pages.order.eta");
  const doneText = useContent("pages.order.done");
  const first = order.items[0];
  const line = etaLine(order, etaText, doneText);
  return (
    <section className="grid gap-x-8 gap-y-10 pb-10 pt-8 lg:grid-cols-12 lg:pt-10">
      <div className="min-w-0 lg:col-span-6 lg:col-start-1">
        <p className="text-[0.6875rem] font-medium uppercase tracking-[0.14em] text-ink-soft">
          {fill(numberText, { number: order.number, date: formatDateTime(vietnamTime(order.placed_at)) })}
        </p>
        <h1 data-testid="order-status" className="mt-4 max-w-[18ch] text-[length:var(--size-h1)] leading-[1.02] text-ink">
          {placed ? thanks : headlines?.[order.status] ?? ""}
        </h1>
        {line && <p className="mt-5 text-sm text-ink-soft">{line}</p>}
      </div>
      {first?.image && (
        <Bleed side="right" className="lg:col-span-5 lg:col-start-8 lg:row-span-2">
          <Link to={`/shop/${first.product}`} aria-label={first.name} data-record={`products:${first.product}`} className="block">
            <ProductImage src={first.image} alt={first.name} aspect="4/3" eager />
          </Link>
        </Bleed>
      )}
      <div className="min-w-0 lg:col-span-6 lg:col-start-1"><OrderSteps status={order.status} /></div>
    </section>
  );
}
