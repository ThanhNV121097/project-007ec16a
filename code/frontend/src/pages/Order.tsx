import { Link, useParams, useSearchParams } from "react-router-dom";
import { T, useContent } from "../editable";
import { usePageMeta } from "../seo";
import { useCollection, useRecord } from "../data";
import { formatDate, formatDateTime, formatPrice } from "../lib/format";
import { usePlacedOrders } from "../lib/mock/orders";
import { fill, type DeliveryOption, type Order as OrderRecord, type OrderStatus, type PaymentMethod, type Product } from "../lib/shop";
import type { CartLine } from "../lib/cart";
import Bleed from "../components/Bleed";
import Disclosure from "../components/Disclosure";
import OrderSteps from "../components/OrderSteps";
import OutlinePill from "../components/OutlinePill";
import ProductImage from "../components/ProductImage";
import Rule from "../components/Rule";
import SpecList from "../components/SpecList";

interface View {
  number: string;
  status: OrderStatus;
  placed_at: string;
  eta?: string;
  lines: CartLine[];
  total: number;
  delivery: string;
  payment: string;
  paid: boolean;
  customer: string;
  city?: string;
  carrier?: string;
  tracking?: string;
}

/** When it arrives, or that it has: the line under the answer. */
function etaLine(view: View, words: { eta: string; done: string }): string {
  if (view.status === "delivered") return words.done;
  if (view.status === "cancelled" || !view.eta) return "";
  return fill(words.eta, { date: formatDate(view.eta) });
}

/** The answer at the left, and the pair it is about running off the right edge. */
function OrderHead({ view, headline, words }: { view: View; headline: string; words: { number: string; eta: string; done: string } }) {
  const first = view.lines[0];
  return (
    <section className="grid gap-x-8 gap-y-10 pb-10 pt-8 lg:grid-cols-12 lg:pt-10">
      <div className="lg:col-span-6">
        <p className="text-[0.6875rem] font-medium uppercase tracking-[0.14em] text-ink-soft">
          {fill(words.number, { number: view.number, date: formatDateTime(view.placed_at) })}
        </p>
        <h1 data-testid="order-status" className="mt-4 max-w-[18ch] text-[length:var(--size-h1)] leading-[1.02] text-ink">{headline}</h1>
        <p className="mt-5 text-sm text-ink-soft">{etaLine(view, words)}</p>
      </div>
      {first?.image && (
        <Bleed side="right" className="lg:col-span-5 lg:col-start-8 lg:row-span-2">
          <Link to={`/shop/${first.id}`} aria-label={first.name} data-record={`products:${first.id}`} className="block">
            <ProductImage src={first.image} alt={first.name} aspect="4/3" eager />
          </Link>
        </Bleed>
      )}
      <div className="lg:col-span-6"><OrderSteps status={view.status} /></div>
    </section>
  );
}

/**
 * AFTER THE ORDER: the answer first and largest — placed, on its way,
 * delivered — then the five steps as one line, then the pairs with their
 * tiles, where they are going and who to ask. The same page answers a buyer
 * straight from checkout and one who looked their order up.
 */
export default function Order() {
  usePageMeta("pages.order.title", "pages.order.description");
  const { id } = useParams();
  const [params] = useSearchParams();
  const record = useRecord<OrderRecord>("orders", id).item;
  const placed = usePlacedOrders().find((o) => o.id === id);
  const product = useRecord<Product>("products", record?.product).item;
  const payments = useCollection<PaymentMethod>("payments", { pageSize: 10 }).items;
  const deliveries = useCollection<DeliveryOption>("delivery", { pageSize: 10 }).items;
  const headlines = useContent<Record<string, string>>("pages.order.headline");
  const thanks = useContent("pages.order.thanks");
  const numberText = useContent("pages.order.number");
  const etaText = useContent("pages.order.eta");
  const doneText = useContent("pages.order.done");
  const paidText = useContent("pages.order.paid");
  const dueText = useContent("pages.order.due");
  const lineText = useContent("pages.checkout.line");
  const spec = useContent<Record<string, string>>("pages.order.spec");

  const view: View | null = placed
    ? { ...placed }
    : record
      ? {
        number: record.number, status: record.status, placed_at: record.placed_at, eta: record.eta,
        lines: product ? [{ id: product.id, name: product.name, price: product.price, image: product.image, variant: `EU ${record.size}`, qty: record.qty }] : [],
        total: record.total, delivery: record.delivery, payment: record.payment, paid: record.paid,
        customer: record.customer, city: record.city, carrier: record.carrier, tracking: record.tracking,
      }
      : null;

  if (!view) {
    return (
      <section className="flex flex-col items-start gap-5 py-16">
        <T k="pages.order.missing" as="h1" className="max-w-[16ch] text-[length:var(--size-h2)] text-ink" />
        <T k="pages.order.missing_line" as="p" className="max-w-[44ch] text-sm text-ink-soft" />
        <OutlinePill to="/track"><T k="pages.order.track_cta" /></OutlinePill>
      </section>
    );
  }

  const headline = params.has("placed") ? thanks : headlines?.[view.status] ?? "";
  const method = payments.find((m) => m.id === view.payment);
  const option = deliveries.find((d) => d.id === view.delivery);

  return (
    <>
      <OrderHead view={view} headline={headline} words={{ number: numberText, eta: etaText, done: doneText }} />
      <Rule />
      <div className="grid gap-12 py-10 lg:grid-cols-12 lg:gap-8">
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
              [spec?.delivery, option ? `${option.name} · ${option.days}` : view.delivery],
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
      </div>
    </>
  );
}
