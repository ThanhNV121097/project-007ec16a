import { T, useContent, useList } from "../editable";
import { usePageMeta } from "../seo";
import { useRecord } from "../data";
import { formatPrice } from "../lib/format";
import { fill, type ShopTerms } from "../lib/shop";
import FaqRows from "../components/FaqRows";
import Icon from "../components/Icon";
import OutlinePill from "../components/OutlinePill";
import Rule from "../components/Rule";

/**
 * RETURNS AND SIZE EXCHANGE: the promise set as three figures on a hairline
 * — days to exchange, days to return, what the first exchange costs — then
 * the three steps, the conditions, and the questions. Written for somebody
 * afraid the size will be wrong, which is most people buying shoes online.
 */
export default function Returns() {
  usePageMeta("pages.returns.title", "pages.returns.description");
  const terms = useRecord<ShopTerms>("shop", "shop").item;
  const fig = useContent<Record<string, string>>("pages.returns.figures");
  const steps = useList<{ title: string; text: string }>("pages.returns.steps");
  const rules = useList<{ text: string }>("pages.returns.rules");
  const values = { n: terms?.exchange_days ?? "", refund: terms?.refund_days ?? "" };
  const figures = [
    { big: String(terms?.exchange_days ?? ""), line: fill(fig?.exchange ?? "", values) },
    { big: String(terms?.refund_days ?? ""), line: fill(fig?.refund ?? "", values) },
    { big: formatPrice(0), line: fill(fig?.free ?? "", values) },
  ];
  return (
    <>
      <section className="grid gap-6 pb-10 pt-8 lg:grid-cols-12 lg:pt-10">
        <T k="pages.returns.heading" as="h1" className="max-w-[12ch] text-[length:var(--size-h1)] leading-[1.02] text-ink lg:col-span-6" />
        <T k="pages.returns.lead" as="p" className="max-w-[44ch] self-end text-sm leading-relaxed text-ink-soft lg:col-span-5 lg:col-start-8" />
      </section>
      <Rule />
      <section className="grid border-b border-line sm:grid-cols-3">
        {figures.map((f, i) => (
          <div key={i} className={`py-10 ${i > 0 ? "border-t border-line sm:border-l sm:border-t-0 sm:pl-8" : "sm:pr-8"}`}>
            <p data-count={i < 2 ? "" : undefined} className="text-[length:var(--size-h1)] leading-none tabular-nums tracking-tight text-ink">{f.big}</p>
            <p className="mt-4 max-w-[28ch] text-sm leading-relaxed text-ink-soft">{f.line}</p>
          </div>
        ))}
      </section>
      <section className="grid gap-8 py-12 lg:grid-cols-12">
        <div className="lg:col-span-4">
          <T k="pages.returns.steps_heading" as="h2" className="text-[length:var(--size-h2)] leading-[1.1] text-ink" />
        </div>
        <ol className="grid gap-8 sm:grid-cols-3 lg:col-span-8">
          {steps.map((s, i) => (
            <li key={i} className="border-t border-ink pt-4">
              <span className="text-sm tabular-nums text-ink-soft">{String(i + 1).padStart(2, "0")}</span>
              <T k={`pages.returns.steps.${i}.title`} as="p" className="mt-3 text-base text-ink" />
              <T k={`pages.returns.steps.${i}.text`} as="p" className="mt-2 text-sm leading-relaxed text-ink-soft" />
            </li>
          ))}
        </ol>
      </section>
      <Rule />
      <section className="grid gap-8 py-12 lg:grid-cols-12">
        <div className="lg:col-span-4">
          <T k="pages.returns.rules_heading" as="h2" className="text-[length:var(--size-h2)] leading-[1.1] text-ink" />
        </div>
        <ul className="border-t border-line lg:col-span-8">
          {rules.map((r, i) => (
            <li key={i} className="flex gap-3 border-b border-line py-4 text-sm text-ink">
              <Icon name="check" size={18} className="mt-0.5 shrink-0" />
              <T k={`pages.returns.rules.${i}.text`} className="max-w-[60ch] leading-relaxed" />
            </li>
          ))}
        </ul>
      </section>
      <Rule />
      <section className="grid gap-8 py-12 lg:grid-cols-12">
        <div className="flex flex-col items-start gap-6 lg:col-span-4">
          <T k="pages.returns.faq_heading" as="h2" className="text-[length:var(--size-h2)] leading-[1.1] text-ink" />
          <OutlinePill to="/contact"><T k="pages.returns.contact_cta" /></OutlinePill>
        </div>
        <FaqRows topic="returns" className="lg:col-span-8" />
      </section>
    </>
  );
}
