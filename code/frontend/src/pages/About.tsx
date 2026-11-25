import { T, useContent } from "../editable";
import { usePageMeta } from "../seo";
import { useCollection, useRecord } from "../data";
import { Picture } from "../picture";
import { fill, type Line, type Product, type ShopTerms } from "../lib/shop";
import OutlinePill from "../components/OutlinePill";
import Rule from "../components/Rule";

/**
 * THE SHOP, IN ITS OWN WORDS — the template's second board: one statement
 * set large and to the right, then two photographs each with a short
 * paragraph beside it, then three figures that are true of the shop today.
 */
export default function About() {
  usePageMeta("pages.about.title", "pages.about.description");
  const pairs = useCollection<Product>("products", { where: { status: "live" }, pageSize: 1 }).total;
  const lines = useCollection<Line>("lines", { pageSize: 1 }).total;
  const terms = useRecord<ShopTerms>("shop", "shop").item;
  const fig = useContent<Record<string, string>>("pages.about.figures");
  const figures = [
    { big: String(pairs), line: fig?.pairs ?? "", count: true },
    { big: String(lines), line: fig?.lines ?? "", count: true },
    { big: String(terms?.rating ?? ""), line: fill(fig?.rating ?? "", { n: terms?.review_count ?? "" }), count: false },
  ];
  return (
    <>
      <section className="flex justify-end pb-16 pt-12 lg:pb-24 lg:pt-20">
        <T k="pages.about.statement" as="h1"
          className="max-w-[20ch] text-right text-[length:var(--size-h1)] leading-[1.04] text-ink" />
      </section>
      <Rule />
      <section className="grid gap-10 py-16 lg:grid-cols-12 lg:items-end lg:gap-8">
        <div className="aspect-[4/3] overflow-hidden bg-surface lg:col-span-7">
          <Picture src="/images/about-stockroom.jpg" alt={useContent("pages.about.first_alt")} width={1280} height={714} motion="parallax" className="h-full w-full object-cover" />
        </div>
        <T k="pages.about.first" as="p" className="max-w-[40ch] text-sm leading-relaxed text-ink-soft lg:col-span-4 lg:col-start-9" />
      </section>
      <section className="grid gap-10 pb-16 lg:grid-cols-12 lg:items-end lg:gap-8">
        <div className="order-2 flex flex-col items-start gap-8 lg:order-1 lg:col-span-6">
          <T k="pages.about.second" as="p" className="max-w-[40ch] text-sm leading-relaxed text-ink-soft" />
          <OutlinePill to="/shop"><T k="pages.about.cta" /></OutlinePill>
        </div>
        <div className="order-1 aspect-[3/4] overflow-hidden bg-surface lg:order-2 lg:col-span-4 lg:col-start-9">
          <Picture src="/images/about-lacing.jpg" alt={useContent("pages.about.second_alt")} width={853} height={1280} className="h-full w-full object-cover" />
        </div>
      </section>
      <Rule />
      <section className="grid sm:grid-cols-3">
        {figures.map((f, i) => (
          <div key={i} className={`py-12 ${i > 0 ? "border-t border-line sm:border-l sm:border-t-0 sm:pl-8" : "sm:pr-8"}`}>
            <p data-count={f.count ? "" : undefined} className="text-[length:var(--size-h1)] leading-none tabular-nums tracking-tight text-ink">{f.big}</p>
            <p className="mt-4 max-w-[28ch] text-sm leading-relaxed text-ink-soft">{f.line}</p>
          </div>
        ))}
      </section>
    </>
  );
}
