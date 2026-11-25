import { collections, schema } from "../../data";
import { sizesInStock, type Category, type Line, type Product } from "../shop";
import type { Page, PriceRange, ProductFilters, ProductsRequest, Sort } from "../shop-request";

/**
 * THE API'S ANSWERS, FROM THE DESIGN'S OWN RECORDS. Deleted in the backend
 * step: `GET /v1/products` and `GET /v1/product-filters` replace the two
 * functions, and src/lib/shop-list.ts is the only file that imports this one.
 *
 * It answers as the story says the API does: every typed word is found in a
 * searched field (accents, case and the stroked d ignored, the line's and the
 * category's name included), "any of" within a filter and "and" between them,
 * an EU size matches only a pair that has it and has not sold it out, equal
 * values keep the records' order, `total` is every match.
 */
function fold(v: unknown): string {
  return String(v ?? "").normalize("NFD").replace(/\p{M}/gu, "").toLowerCase().replace(/đ/g, "d");
}

function table<T>(name: string): T[] {
  return (collections()[name] ?? []) as unknown as T[];
}

function later<T>(value: T): Promise<T> {
  return new Promise((resolve) => {
    setTimeout(() => resolve(value), 0);
  });
}

function inAny(list: readonly string[], value: string | undefined): boolean {
  return list.length === 0 || (value !== undefined && list.includes(value));
}

function inRanges(price: number, ranges: readonly PriceRange[]): boolean {
  return ranges.length === 0 || ranges.some((r) => (r.from === undefined || price >= r.from) && (r.to === undefined || price <= r.to));
}

function hasWords(p: Product, q: string): boolean {
  const words = fold(q).split(/\s+/).filter(Boolean);
  if (words.length === 0) return true;
  const line = table<Line>("lines").find((l) => l.id === p.line)?.name;
  const category = table<Category>("categories").find((c) => c.id === p.category)?.name;
  const text = fold([p.name, p.model, p.colourway, p.code, line, category, p.description].join("\n"));
  return words.every((w) => text.includes(w));
}

function matches(p: Product, req: ProductsRequest): boolean {
  const available = sizesInStock(p);
  return p.status === req.status
    && inAny(req.category, p.category)
    && inAny(req.line, p.line)
    && inAny(req.colour, p.colour)
    && inRanges(p.price, req.price)
    && (req.size.length === 0 || req.size.some((s) => available.includes(s)))
    && hasWords(p, req.q);
}

/** A shoe with no rating comes last on "Best rated". */
const ORDER: Record<Sort, (a: Product, b: Product) => number> = {
  "-released": (a, b) => (b.released ?? "").localeCompare(a.released ?? ""),
  price: (a, b) => a.price - b.price,
  "-price": (a, b) => b.price - a.price,
  "-rating": (a, b) => (b.rating ?? 0) - (a.rating ?? 0),
};

/** `GET /v1/products?status=live&limit=12&page=n&...` */
export function listProducts(req: ProductsRequest): Promise<Page<Product>> {
  const hits = table<Product>("products").filter((p) => matches(p, req)).sort(ORDER[req.sort]);
  const start = (req.page - 1) * req.limit;
  return later({ items: hits.slice(start, start + req.limit), total: hits.length, page: req.page, limit: req.limit });
}

/** "36 2/3" as a number, so sizes sort the way a foot grows. */
function sizeValue(s: string): number {
  const [whole, frac] = s.split(" ");
  if (!frac) return Number(whole);
  const [a, b] = frac.split("/").map(Number);
  return Number(whole) + (b ? a / b : 0);
}

/** The colour set's own order: black, white, pink, ... */
function colourOrder(): string[] {
  const options = schema.collections.products?.fields.find((f) => f.key === "colour")?.options ?? [];
  return options.map((o) => (typeof o === "string" ? o : o[0]));
}

/** `GET /v1/product-filters`: every colour and size of the live shoes, whatever is filtered. */
export function getProductFilters(): Promise<ProductFilters> {
  const live = table<Product>("products").filter((p) => p.status === "live");
  const have = new Set(live.map((p) => p.colour));
  return later({
    colours: colourOrder().filter((c) => have.has(c)),
    sizes: [...new Set(live.flatMap((p) => p.sizes ?? []))].sort((a, b) => sizeValue(a) - sizeValue(b)),
  });
}
