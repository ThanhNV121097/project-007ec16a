import { T } from "../editable";
import { usePageMeta } from "../seo";
import { Picture } from "../picture";
import OutlinePill from "../components/OutlinePill";
import Rule from "../components/Rule";

/**
 * The shop, in its own words: one statement set large and to the right, the
 * way the template's second board sets it, then two photographs each with a
 * short paragraph beside it.
 *
 * The two pictures are the only scenes on the whole site. They are the
 * owner's own wherever the owner has them — a shop front, a workbench — and
 * a paragraph with no picture for it loses the picture, not the other way
 * round.
 */
export default function About() {
  usePageMeta("pages.about.title", "pages.about.description");
  return (
    <>
      <section className="flex justify-end py-16 lg:py-24">
        <T k="pages.about.statement" as="h1"
          className="max-w-[20ch] text-right text-[length:var(--size-h1)] leading-[1.04] text-ink" />
      </section>
      <Rule />
      <section className="flex flex-col gap-10 py-16 lg:flex-row lg:items-end lg:gap-16">
        <div className="lg:w-7/12">
          <Picture src="/images/about-1.jpg" alt="The shop at work" width={1200} height={900} className="aspect-[4/3] w-full" />
        </div>
        <T k="pages.about.first" as="p" className="max-w-[36ch] text-sm leading-relaxed text-ink-soft lg:w-5/12 lg:pb-2" />
      </section>
      <section className="flex flex-col-reverse gap-10 pb-16 lg:flex-row lg:items-end lg:gap-16">
        <div className="flex flex-col items-start gap-8 lg:w-7/12 lg:pb-2">
          <T k="pages.about.second" as="p" className="max-w-[36ch] text-sm leading-relaxed text-ink-soft" />
          <OutlinePill to="/shop"><T k="pages.about.cta" /></OutlinePill>
        </div>
        <div className="lg:w-5/12">
          <Picture src="/images/about-2.jpg" alt="A detail of the work" width={900} height={1200} className="aspect-[3/4] w-full" />
        </div>
      </section>
    </>
  );
}
