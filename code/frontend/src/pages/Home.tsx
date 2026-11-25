import { T } from "../editable";
import { usePageMeta } from "../seo";
import { useCollection } from "../data";
import { formatPrice } from "../lib/format";
import type { Product } from "../lib/shop";
import Grid from "../components/Grid";
import OutlinePill from "../components/OutlinePill";
import ProductCard from "../components/ProductCard";
import ProductImage from "../components/ProductImage";
import Rule from "../components/Rule";

/**
 * The home page: ONE thing at full size, then a row of others, then one
 * sentence about the shop. No photograph across the top — on this look the
 * first screen is the product itself, its name large at the left and its
 * tile filling the right half.
 *
 * The featured product is the record marked `featured` in src/data, so what
 * opens the site is a choice the owner makes in the Data panel, not a line
 * of code.
 */
export default function Home() {
  usePageMeta("pages.home.title", "pages.home.description");
  const featured = useCollection<Product>("products", { where: { featured: true }, pageSize: 1 }).items[0];
  const { items } = useCollection<Product>("products", { pageSize: 5 });
  const lead = featured ?? items[0];
  const more = items.filter((p) => p.id !== lead?.id).slice(0, 4);
  return (
    <>
      {lead && (
        <section data-record={`products:${lead.id}`} className="flex flex-col gap-10 py-10 lg:flex-row lg:items-stretch lg:gap-16">
          <div className="flex flex-col lg:w-5/12 lg:py-4">
            <T k="pages.home.eyebrow" as="p" className="text-[0.6875rem] font-medium uppercase tracking-[0.14em] text-ink-soft" />
            <h1 className="mt-4 max-w-[11ch] text-[length:var(--size-h1)] leading-[1.02] text-ink">{lead.name}</h1>
            <p className="mt-5 text-lg tabular-nums text-ink">{formatPrice(lead.price)}</p>
            {lead.description && <p className="mt-6 max-w-[34ch] text-sm leading-relaxed text-ink-soft">{lead.description}</p>}
            <div className="mt-8 lg:mt-auto" data-magnetic="">
              <OutlinePill to={`/shop/${lead.id}`} testid="home-lead"><T k="pages.home.lead_cta" /></OutlinePill>
            </div>
          </div>
          <div className="lg:w-7/12" data-tilt="">
            <ProductImage src={lead.image} alt={lead.name} aspect="4/3" eager />
          </div>
        </section>
      )}

      <Rule />
      <div className="flex items-center justify-between gap-4 py-8">
        <T k="pages.home.more" as="h2" className="text-xl text-ink" />
        <OutlinePill to="/shop" testid="home-more"><T k="pages.home.more_cta" /></OutlinePill>
      </div>
      <Rule />
      <section className="py-10">
        <Grid max={4}>
          {more.map((p) => <ProductCard key={p.id} product={p} />)}
        </Grid>
      </section>

      <Rule />
      <section className="flex flex-col items-end gap-8 py-20">
        <T k="pages.home.statement" as="p"
          className="max-w-[22ch] text-right font-display text-[length:var(--size-h2)] leading-[1.1] tracking-tight text-ink lg:text-[2.75rem]" />
        <OutlinePill to="/about"><T k="pages.home.statement_cta" /></OutlinePill>
      </section>
    </>
  );
}
