import { useState } from "react";
import { T, useContent, useList } from "../editable";
import { usePageMeta } from "../seo";
import { useCollection } from "../data";
import { fill, type Line, type SizeRow } from "../lib/shop";
import Bleed from "../components/Bleed";
import Disclosure from "../components/Disclosure";
import FaqRows from "../components/FaqRows";
import Rule from "../components/Rule";

const LABEL = "mb-1.5 block text-[0.6875rem] font-medium uppercase tracking-[0.12em] text-ink-soft";

/**
 * FIND YOUR SIZE: three steps to measure a foot and a box that answers with
 * the EU size, beside the whole adidas chart in hairline rows — numbers
 * right-aligned, the answered row marked. Then how each line fits, and the
 * questions customers ask about size.
 */
export default function SizeChart() {
  usePageMeta("pages.sizes.title", "pages.sizes.description");
  const rows = useCollection<SizeRow>("sizes", { pageSize: 60 }).items;
  const lines = useCollection<Line>("lines", { pageSize: 30 }).items;
  const steps = useList<{ text: string }>("pages.sizes.steps");
  const answerText = useContent("pages.sizes.answer_line");
  const tooBig = useContent("pages.sizes.too_big");
  const sizeText = useContent("pages.sizes.answer_size");
  const head = useContent<Record<string, string>>("pages.sizes.columns");
  const [cm, setCm] = useState("");
  const length = parseFloat(cm.replace(",", "."));
  const asked = Number.isFinite(length) && length >= 15 && length <= 35;
  const answer = asked ? rows.find((r) => r.cm >= length) : undefined;

  return (
    <>
      <section className="grid gap-12 pb-12 pt-8 lg:grid-cols-12 lg:gap-8 lg:pt-10">
        <div className="lg:col-span-5">
          <T k="pages.sizes.heading" as="h1" className="max-w-[10ch] text-[length:var(--size-h1)] leading-[1.02] text-ink" />
          <T k="pages.sizes.lead" as="p" className="mt-5 max-w-[40ch] text-sm leading-relaxed text-ink-soft" />
          <ol className="mt-8 border-t border-line">
            {steps.map((s, i) => (
              <li key={i} className="flex gap-4 border-b border-line py-4 text-sm text-ink">
                <span className="w-6 shrink-0 tabular-nums text-ink-soft">{String(i + 1).padStart(2, "0")}</span>
                <T k={`pages.sizes.steps.${i}.text`} className="max-w-[44ch] leading-relaxed" />
              </li>
            ))}
          </ol>
          <div className="mt-8">
            <label htmlFor="foot-length" className={LABEL}><T k="pages.sizes.input" /></label>
            <div className="flex items-center gap-3">
              <input id="foot-length" inputMode="decimal" value={cm} onChange={(e) => setCm(e.target.value)} placeholder="24.5"
                className="w-32 appearance-none rounded-sm border border-field bg-ground px-3 py-3 text-base tabular-nums text-ink outline-none placeholder:text-ink-soft focus:border-ink" />
              <span className="text-sm text-ink-soft">{head?.cm}</span>
            </div>
            <div data-testid="size-answer" aria-live="polite" className="mt-6 min-h-24">
              {answer ? (
                <>
                  <p className="text-[length:var(--size-h2)] leading-none tabular-nums text-ink">{fill(sizeText, { eu: answer.eu })}</p>
                  <p className="mt-2 text-sm tabular-nums text-ink-soft">{fill(answerText, { uk: answer.uk, us_m: answer.us_m, us_w: answer.us_w })}</p>
                </>
              ) : asked ? (
                <p className="text-sm text-ink-soft">{tooBig}</p>
              ) : (
                <T k="pages.sizes.hint" as="p" className="text-sm text-ink-soft" />
              )}
            </div>
          </div>
        </div>
        <Bleed side="right" ground className="py-8 lg:col-span-6 lg:col-start-7">
          <table className="w-full text-sm">
            <caption className="pb-4 text-left text-lg text-ink"><T k="pages.sizes.table" /></caption>
            <thead>
              <tr className="border-y border-line text-[0.6875rem] uppercase tracking-[0.12em] text-ink-soft">
                <th scope="col" className="py-3 text-left font-medium">{head?.eu}</th>
                <th scope="col" className="py-3 text-right font-medium">{head?.uk}</th>
                <th scope="col" className="py-3 text-right font-medium">{head?.us_m}</th>
                <th scope="col" className="py-3 text-right font-medium">{head?.us_w}</th>
                <th scope="col" className="py-3 text-right font-medium">{head?.cm}</th>
              </tr>
            </thead>
            <tbody className="tabular-nums">
              {rows.map((r) => {
                const on = answer?.id === r.id;
                return (
                  <tr key={r.id} data-record={`sizes:${r.id}`} className={`border-b border-line ${on ? "bg-wash" : ""}`}>
                    <th scope="row" className={`py-2.5 pl-1 text-left font-normal ${on ? "text-ink" : "text-ink"}`}>{r.eu}</th>
                    <td className="py-2.5 text-right text-ink-soft">{r.uk}</td>
                    <td className="py-2.5 text-right text-ink-soft">{r.us_m}</td>
                    <td className="py-2.5 text-right text-ink-soft">{r.us_w}</td>
                    <td className="py-2.5 pr-1 text-right text-ink">{r.cm.toFixed(1)}</td>
                  </tr>
                );
              })}
            </tbody>
          </table>
          <T k="pages.sizes.note" as="p" className="mt-4 max-w-[60ch] text-xs leading-relaxed text-ink-soft" />
        </Bleed>
      </section>
      <Rule />
      <section className="grid gap-8 py-12 lg:grid-cols-12">
        <div className="lg:col-span-4">
          <T k="pages.sizes.fit_heading" as="h2" className="text-[length:var(--size-h2)] leading-[1.1] text-ink" />
        </div>
        <div className="lg:col-span-8">
          <Disclosure rows={lines.map((l) => ({ label: l.name, content: <p className="max-w-[56ch] leading-relaxed">{l.fit_note}</p> }))} />
        </div>
      </section>
      <Rule />
      <section className="grid gap-8 py-12 lg:grid-cols-12">
        <div className="lg:col-span-4">
          <T k="pages.sizes.faq_heading" as="h2" className="text-[length:var(--size-h2)] leading-[1.1] text-ink" />
        </div>
        <FaqRows topic="sizing" className="lg:col-span-8" />
      </section>
    </>
  );
}
