import { T, useContent } from "../editable";
import { usePageMeta } from "../seo";
import { useCollection, useRecord } from "../data";
import type { DeliveryOption, ShopTerms } from "../lib/shop";
import HeadPicture from "../components/HeadPicture";
import FaqRows from "../components/FaqRows";
import OutlinePill from "../components/OutlinePill";
import Rule from "../components/Rule";

/**
 * DELIVERY: the two things a buyer asks — does it cost anything, how long
 * does it take — set as oversized figures on a hairline, then the options as
 * a table with the cost right-aligned, then the questions. Delivery is free
 * on every order, so the cost is the bag's own word for it (pages.cart.free)
 * and the days are the shop's own term in src/data, so the owner changes
 * each once.
 */
export default function Shipping() {
  usePageMeta("pages.shipping.title", "pages.shipping.description");
  const terms = useRecord<ShopTerms>("shop", "shop").item;
  const options = useCollection<DeliveryOption>("delivery", { pageSize: 10 }).items;
  const free = useContent("pages.cart.free");
  const fig = useContent<Record<string, string>>("pages.shipping.figures");
  const cols = useContent<Record<string, string>>("pages.shipping.columns");
  const figures = [
    { big: free, line: fig?.free ?? "" },
    { big: terms?.standard_days ?? "", line: fig?.standard ?? "" },
  ];
  return (
    <>
      <section className="grid gap-6 pb-10 pt-8 lg:grid-cols-12 lg:gap-8 lg:pt-10">
        <div className="flex flex-col justify-end lg:col-span-6">
          <T k="pages.shipping.heading" as="h1" className="text-[length:var(--size-h1)] leading-[1.02] text-ink" />
          <T k="pages.shipping.lead" as="p" className="mt-5 max-w-[44ch] text-sm leading-relaxed text-ink-soft" />
        </div>
        <HeadPicture src="/images/head-delivery.jpg" alt={useContent("pages.shipping.head_alt")} className="lg:col-span-6 lg:col-start-7" />
      </section>
      <Rule />
      <section className="grid border-b border-line sm:grid-cols-2">
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
        </div>
        <div className="overflow-x-auto lg:col-span-8">
          <table className="w-full min-w-[32rem] text-sm">
            <thead>
              <tr className="border-y border-line text-[0.6875rem] uppercase tracking-[0.12em] text-ink-soft">
                <th scope="col" className="py-3 text-left font-medium">{cols?.option}</th>
                <th scope="col" className="py-3 text-left font-medium">{cols?.time}</th>
                <th scope="col" className="py-3 text-left font-medium">{cols?.where}</th>
                <th scope="col" className="py-3 text-right font-medium">{cols?.cost}</th>
              </tr>
            </thead>
            <tbody>
              {options.map((o) => (
                <tr key={o.id} data-record={`delivery:${o.id}`} className="border-b border-line align-top">
                  <th scope="row" className="py-4 pr-4 text-left font-normal text-ink">{o.name}</th>
                  <td className="py-4 pr-4 text-ink">{o.days}</td>
                  <td className="max-w-[36ch] py-4 pr-4 text-ink-soft">{o.note}</td>
                  <td className="py-4 text-right tabular-nums text-ink">{free}</td>
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
