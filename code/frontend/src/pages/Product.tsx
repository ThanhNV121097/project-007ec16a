import { useState } from "react";
import { Link, useParams } from "react-router-dom";
import { T } from "../editable";
import { useCollection, useRecord } from "../data";
import { formatPrice } from "../lib/format";
import { picturesOf, type Kind, type Product as Item } from "../lib/shop";
import ColourDots from "../components/ColourDots";
import Disclosure from "../components/Disclosure";
import Icon from "../components/Icon";
import OutlinePill from "../components/OutlinePill";
import ProductCard from "../components/ProductCard";
import ProductGallery from "../components/ProductGallery";
import RelatedRow from "../components/RelatedRow";
import Rule from "../components/Rule";
import SpecList from "../components/SpecList";
import VariantPicker from "../components/VariantPicker";

/**
 * One product, and everything somebody deciding about it asks.
 *
 * This is the page the template's own picture shows: the name large at the
 * left, the tile at the right, and the rest behind three disclosure rows so
 * the page stays a page and not a column of paragraphs. Each part draws
 * nothing when the record gives it nothing — a product with no care note has
 * no Care row — so what is on the page is decided in src/data.
 */
export default function Product() {
  const { id } = useParams();
  const { item } = useRecord<Item>("products", id);
  const kinds = useCollection<Kind>("categories", { pageSize: 50 }).items;
  const [colour, setColour] = useState("");
  const [size, setSize] = useState("");
  if (!item) {
    return (
      <div className="flex flex-col items-start gap-6 py-24">
        <T k="pages.product.missing" as="h1" className="text-[length:var(--size-h2)] text-ink" />
        <OutlinePill to="/shop"><T k="pages.product.back" /></OutlinePill>
      </div>
    );
  }
  const kind = kinds.find((k) => k.id === item.category)?.name;
  return (
    <>
      <section className="flex flex-col gap-10 py-10 lg:flex-row lg:gap-16">
        <div data-record={`products:${item.id}`} className="flex flex-col gap-7 lg:w-5/12">
          <Link to="/shop" className="inline-flex items-center gap-1.5 self-start text-[0.6875rem] font-medium uppercase tracking-[0.12em] text-ink-soft hover:text-ink">
            <Icon name="back" size={14} /><T k="pages.product.back" />
          </Link>
          <div>
            <h1 data-testid="product-name" className="max-w-[11ch] text-[length:var(--size-h1)] leading-[1.02] text-ink">{item.name}</h1>
            <p className="mt-5 text-lg tabular-nums text-ink">{formatPrice(item.price)}</p>
          </div>
          <ColourDots label={<T k="pages.product.colour" />} colours={item.colours ?? []} value={colour || item.colours?.[0] || ""} onChange={setColour} />
          <VariantPicker label={<T k="pages.product.size" />} options={item.sizes ?? []} value={size} onChange={setSize} />
          {item.description && <p className="max-w-[40ch] text-sm leading-relaxed text-ink-soft">{item.description}</p>}
          <div>
            <OutlinePill to={`/contact?about=${encodeURIComponent(item.name)}`} testid="product-ask">
              <T k="pages.product.ask" />
            </OutlinePill>
          </div>
          <Disclosure rows={[
            { label: <T k="pages.product.details" />, content: (
              <SpecList items={[
                [<T key="k" k="pages.product.spec_kind" />, kind],
                [<T key="m" k="pages.product.spec_material" />, item.material],
                [<T key="s" k="pages.product.spec_sizes" />, item.sizes?.join(", ")],
                [<T key="c" k="pages.product.spec_colours" />, item.colours?.join(", ")],
              ]} />
            ) },
            { label: <T k="pages.product.care" />, content: item.care },
            { label: <T k="pages.product.delivery" />, content: <T k="pages.product.delivery_text" /> },
          ]} />
        </div>
        <div className="lg:w-7/12">
          <ProductGallery images={picturesOf(item)} alt={item.name} aspect="4/5" />
        </div>
      </section>

      <Rule />
      <RelatedRow<Item> collection="products" exclude={item.id} where={{ category: item.category }} className="py-10"
        title={<T k="pages.product.related" as="h2" className="text-xl text-ink" />}>
        {(p) => <ProductCard product={p} />}
      </RelatedRow>
    </>
  );
}
