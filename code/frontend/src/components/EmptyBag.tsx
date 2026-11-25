import { T } from "../editable";
import { useCollection } from "../data";
import type { Product } from "../lib/shop";
import Grid from "./Grid";
import OutlinePill from "./OutlinePill";
import ProductCard from "./ProductCard";
import Rule from "./Rule";

/**
 * AN EMPTY BAG IS A PAGE TOO: it says so at display size, says what to do,
 * and puts this month's pairs under a hairline so the next step is a tap.
 * The bag and the checkout both show it when there is nothing to pay for.
 */
export default function EmptyBag({ heading = "pages.cart.empty_heading" }: { heading?: string }) {
  const fresh = useCollection<Product>("products", { where: { status: "live", is_new: true }, pageSize: 4 }).items;
  return (
    <>
      <section className="flex flex-col items-start gap-5 pb-12 pt-10">
        <T k={heading} as="h1" className="max-w-[16ch] text-[length:var(--size-h1)] leading-[1.02] text-ink" />
        <T k="pages.cart.empty_line" as="p" className="max-w-[44ch] text-sm leading-relaxed text-ink-soft" />
        <OutlinePill to="/shop"><T k="pages.cart.keep" /></OutlinePill>
      </section>
      <Rule />
      <div className="py-7"><T k="pages.cart.suggest" as="h2" className="text-lg text-ink" /></div>
      <Rule />
      <section className="py-10">
        <Grid count={fresh.length} max={4}>{fresh.map((p) => <ProductCard key={p.id} product={p} />)}</Grid>
      </section>
    </>
  );
}
