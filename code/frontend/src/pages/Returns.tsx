import { T, useContent, useList } from "../editable";
import { usePageMeta } from "../seo";
import HeadPicture from "../components/HeadPicture";
import FaqRows from "../components/FaqRows";
import Icon from "../components/Icon";
import OutlinePill from "../components/OutlinePill";
import Rule from "../components/Rule";

/**
 * RETURN POLICY: exchanges are not accepted, said plainly. The owner gave no
 * return window, shoe condition, return-shipping rule or refund terms, so
 * none is stated here. Three words on a hairline, the steps, the short list
 * and the questions; the contact route is where to ask or to report a wrong
 * or damaged delivery.
 */
export default function Returns() {
  usePageMeta("pages.returns.title", "pages.returns.description");
  const figures = useList<{ big: string; line: string }>("pages.returns.figures");
  const steps = useList<{ title: string; text: string }>("pages.returns.steps");
  const rules = useList<{ text: string }>("pages.returns.rules");
  return (
    <>
      <section className="grid gap-6 pb-10 pt-8 lg:grid-cols-12 lg:gap-8 lg:pt-10">
        <div className="flex flex-col justify-end lg:col-span-6">
          <T k="pages.returns.heading" as="h1" className="max-w-[12ch] text-[length:var(--size-h1)] leading-[1.02] text-ink" />
          <T k="pages.returns.lead" as="p" className="mt-5 max-w-[44ch] text-sm leading-relaxed text-ink-soft" />
        </div>
        <HeadPicture src="/images/head-returns.jpg" alt={useContent("pages.returns.head_alt")} className="lg:col-span-6 lg:col-start-7" />
      </section>
      <Rule />
      <section className="grid border-b border-line sm:grid-cols-3">
        {figures.map((f, i) => (
          <div key={f.big} className={`py-10 ${i > 0 ? "border-t border-line sm:border-l sm:border-t-0 sm:pl-8" : "sm:pr-8"}`}>
            <T k={`pages.returns.figures.${i}.big`} as="p" className="text-[length:var(--size-h1)] leading-none tracking-tight text-ink" />
            <T k={`pages.returns.figures.${i}.line`} as="p" className="mt-4 max-w-[28ch] text-sm leading-relaxed text-ink-soft" />
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
