import { Link } from "react-router-dom";
import ProductImage from "./ProductImage";
import { formatPrice } from "../lib/format";
import type { Product } from "../lib/shop";

/**
 * A PRODUCT, AS THIS LOOK SHOWS ONE: the thing alone on a pale grey tile, and
 * under it one caption row — the name in letterspaced capitals at the left,
 * the price at the right. No border, no shadow, no badge, no button: the
 * whole card is the link.
 */
export default function ProductCard({ product }: { product: Product }) {
  return (
    <Link
      to={`/shop/${product.id}`}
      data-record={`products:${product.id}`}
      data-testid={`card-${product.id}`}
      className="group block"
    >
      <ProductImage src={product.image} hover={product.image_back} alt={product.name} aspect="4/5" />
      <span className="mt-3 flex flex-wrap items-baseline justify-between gap-x-3 gap-y-1 text-[0.6875rem] font-medium uppercase tracking-[0.1em] text-ink">
        <span className="min-w-0 underline-offset-4 group-hover:underline">{product.name}</span>
        <span className="shrink-0 tabular-nums">{formatPrice(product.price)}</span>
      </span>
    </Link>
  );
}
