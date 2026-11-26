import { Link } from "react-router-dom";
import ProductImage from "./ProductImage";
import { fill } from "../lib/shop";
import type { LowShoe } from "../lib/overview";

/** What a running-low row needs of a shoe: only what the overview's answer carries. */
export type LowRow = Pick<LowShoe, "id" | "name" | "image" | "stock" | "sizes_low">;

/**
 * What is running low, on the overview: the shoes down to their last pairs,
 * least first, each a link to the product — its picture, its name, the sizes
 * running out and how many pairs are left.
 */
export default function AdminLowStock({ rows, sizesText }: {
  rows: readonly LowRow[];
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
              <span className="block break-words font-medium text-ink">{p.name}</span>
              <span className="block text-ink-soft">{fill(sizesText, { sizes: p.sizes_low.join(", ") })}</span>
            </span>
            <span className="shrink-0 text-sm tabular-nums text-ink">{p.stock}</span>
          </Link>
        </li>
      ))}
    </ul>
  );
}
