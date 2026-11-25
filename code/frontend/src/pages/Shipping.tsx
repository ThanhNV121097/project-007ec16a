import { T, useContent } from "../editable";
import { usePageMeta } from "../seo";
import { useCollection, useRecord } from "../data";
import { formatPrice } from "../lib/format";
import { fill, type DeliveryOption, type ShopTerms } from "../lib/shop";
import FaqRows from "../components/FaqRows";
import OutlinePill from "../components/OutlinePill";
import Rule from "../components/Rule";

/** The three oversized figures, from the shop's own terms. */
function figuresOf(terms: ShopTerms | undefined, fig: Record<string, string>, free: string, values: Record<string, string>) {
  const t = terms ?? ({} as Partial<ShopTerms>);
  return [
    { big: free, line: fill(fig.free ?? "", values) },
    { big: t.standard_days ?? "", line: fill(fig.standard ?? "", values) },
    { big: t.express_days ?? "", line: fill(fig.express ?? "", values) },
  ];
}

/**
 * DELIVERY: the three things a buyer asks — what it costs, how long, how
 * fast it can be — set as oversized figures on a hairline, then the options
 * as a table with the fees right-aligned, then the questions. Every number
 * is the shop's own term in src/data, so the owner changes it once.
 */
export default function Shipping() {
  usePageMeta("pages.shipping.title", "pages.shipping.description");
  const terms = useRecord<ShopTerms>("shop", "shop").item;
  const options = useCollection<DeliveryOption>("delivery", { pageSize: 10 }).items;
  const free = useContent("pages.cart.free");
  const fig = useContent<Record<string, string>>("pages.shipping.figures");
  const cols = useContent<Record<string, string>>("pages.shipping.columns");
  const freeFrom = useContent("pages.shipping.free_from");
  const values = {
    price: formatPrice(terms?.free_delivery_from ?? 0),
    cities: terms?.express_cities ?? "",
    cutoff: terms?.dispatch_cutoff ?? "",
  };
  const figures = figuresOf(terms, fig ?? {}, free, values);
  const cutoffLine = fill(useContent("pages.shipping.cutoff"), values);
  return (
    <>
      <section className="grid gap-6 pb-10 pt-8 lg:grid-cols-12 lg:pt-10">
        <T k="pages.shipping.heading" as="h1" className="text-[length:var(--size-h1)] leading-[1.02] text-ink lg:col-span-6" />
        <T k="pages.shipping.lead" as="p" className="max-w-[44ch] self-end text-sm leading-relaxed text-ink-soft lg:col-span-5 lg:col-start-8" />
      </section>
      <Rule />
      <section className="grid border-b border-line sm:grid-cols-3">
        {figures.map((f, i) => (
          <div key={i} className={`py-10 ${i > 0 ? "border-t border-line sm:border-l sm:border-t-0 sm:pl-8" : "sm:pr-8"}`}>
            <p className="text-[length:var(--size-h1)] leading-none tabular-nums tracking-tight text-ink">{f.big}</p>
            <p className="mt-4 max-w-[28ch] text-sm leading-relaxed text-ink-soft">{f.line}</p>
          </div>
        ))}
      </section>
      <section className="grid gap-8 py-12 lg:grid-cols-12">
        <div className="lg:col-span-4">
          <T k="pages.shipping.options_heading" as="h2" className="text-[length:var(--size-h2)] leading-[1.1] text-ink" />
          <p className="mt-3 max-w-[34ch] text-sm leading-relaxed text-ink-soft">{cutoffLine}</p>
        </div>
        <div className="overflow-x-auto lg:col-span-8">
          <table className="w-full min-w-[32rem] text-sm">
            <thead>
              <tr className="border-y border-line text-[0.6875rem] uppercase tracking-[0.12em] text-ink-soft">
                <th scope="col" className="py-3 text-left font-medium">{cols?.option}</th>
                <th scope="col" className="py-3 text-left font-medium">{cols?.time}</th>
                <th scope="col" className="py-3 text-left font-medium">{cols?.where}</th>
                <th scope="col" className="py-3 text-right font-medium">{cols?.fee}</th>
              </tr>
            </thead>
            <tbody>
              {options.map((o) => (
                <tr key={o.id} data-record={`delivery:${o.id}`} className="border-b border-line align-top">
                  <th scope="row" className="py-4 pr-4 text-left font-normal text-ink">{o.name}</th>
                  <td className="py-4 pr-4 text-ink">{o.days}</td>
                  <td className="max-w-[36ch] py-4 pr-4 text-ink-soft">{o.note}</td>
                  <td className="py-4 text-right tabular-nums text-ink">
                    {formatPrice(o.fee)}
                    {o.id === "standard" && <span className="block text-xs text-ink-soft">{fill(freeFrom, values)}</span>}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>
      <Rule />
      <section className="grid gap-8 py-12 lg:grid-cols-12">
        <div className="flex flex-col items-start gap-6 lg:col-span-4">
          <T k="pages.shipping.faq_heading" as="h2" className="text-[length:var(--size-h2)] leading-[1.1] text-ink" />
          <OutlinePill to="/track"><T k="pages.shipping.track_cta" /></OutlinePill>
        </div>
        <FaqRows topic="shipping" className="lg:col-span-8" />
      </section>
    </>
  );
}
