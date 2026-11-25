import { Link } from "react-router-dom";
import { useContent } from "../editable";
import { useRecord } from "../data";
import ProductImage from "./ProductImage";
import { formatPrice } from "../lib/format";
import { fill, sizesInStock, type Line, type Product } from "../lib/shop";

/**
 * A SHOE, AS THIS LOOK SHOWS ONE: the pair alone on its pale grey tile, and
 * under it one caption row — the name in letterspaced capitals at the left,
 * the price at the right — then one quiet line saying which line it is and
 * how many sizes are left. No border, no shadow, no badge: the card is the
 * link, and the pointer shows the front pair.
 */
export default function ProductCard({ product, tilt = false }: { product: Product; tilt?: boolean }) {
  const line = useRecord<Line>("lines", product.line).item;
  const note = fill(useContent("card.sizes"), { n: sizesInStock(product).length });
  return (
    <Link
      to={`/shop/${product.id}`}
      data-record={`products:${product.id}`}
      data-testid={`card-${product.id}`}
      data-tilt={tilt ? "" : undefined}
      className="group flex h-full flex-col"
    >
      <ProductImage src={product.image} hover={product.image_back} alt={product.name} aspect="1/1"
        slot={`/images/products/${product.id}.jpg`} />
      <span className="mt-3 flex flex-col gap-1 text-[0.8125rem] font-medium uppercase tracking-[0.1em] text-ink sm:flex-row sm:items-baseline sm:justify-between sm:gap-3">
        <span className="min-w-0 underline-offset-4 group-hover:underline">{product.name}</span>
        <span className="shrink-0 tabular-nums">{formatPrice(product.price)}</span>
      </span>
      <span className="mt-1 block text-sm text-ink-soft">{line?.name ? `${line.name} · ${note}` : note}</span>
    </Link>
  );
}
