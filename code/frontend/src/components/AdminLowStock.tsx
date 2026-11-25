import { Link } from "react-router-dom";
import ProductImage from "./ProductImage";
import { fill, type Product } from "../lib/shop";

/**
 * What is running low, on the overview: the shoes down to their last pairs,
 * least first, each a link to the product — its picture, its name, the sizes
 * running out and how many pairs are left.
 */
export default function AdminLowStock({ rows, sizesText }: {
  rows: readonly Product[];
  /** The sizes line, from content.json so the owner can change it; {sizes} is filled in. */
  sizesText: string;
}) {
  return (
    <ul className="divide-y divide-line">
      {rows.map((p) => (
        <li key={p.id}>
          <Link to={`/admin/products/${p.id}`} className="flex items-center gap-3 py-2.5 hover:bg-wash">
            <span className="w-10 shrink-0"><ProductImage src={p.image} alt={p.name} aspect="1/1" /></span>
            <span className="min-w-0 flex-1 text-sm">
              <span className="block font-medium text-ink">{p.name}</span>
              <span className="block text-ink-soft">{fill(sizesText, { sizes: (p.sizes_low ?? []).join(", ") })}</span>
            </span>
            <span className="shrink-0 text-sm tabular-nums text-ink">{p.stock ?? 0}</span>
          </Link>
        </li>
      ))}
    </ul>
  );
}
