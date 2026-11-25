import { schema } from "../../data";
import { api, type Params } from "../api";
import { sizesInStock, type Category, type Line, type Product } from "../shop";
import type { Page, PriceRange, ProductFilters, ProductsRequest, Sort } from "../shop-request";

/**
 * THE API'S ANSWERS, FROM THE SITE'S OWN RECORDS. Deleted in the backend
 * step: `GET /v1/products` and `GET /v1/product-filters` replace the two
 * functions, and src/lib/shop-list.ts is the only file that imports this one.
 *
 * The shoes are read from the API itself (`GET /v1/products?status=live`,
 * every page of them, on every call) and narrowed here, in the browser. They
 * are NOT taken from the rows other pages have already put in src/data's
 * store: that store is empty when /shop is the first page opened, and a list
 * read from it said "no pair matches" for a catalogue that was there. A read
 * that fails rejects, as the real client will, so a refused database, a
 * stopped server or an offline browser is "could not load" and never an empty
 * list.
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

function wait(ms: number): Promise<void> {
  return new Promise((resolve) => {
    setTimeout(resolve, ms);
  });
}

/** The most rows the API serves in one page. */
const CEILING = 100;

/** Every row of a collection, a page after another; a failed page rejects, as `api.get` does. */
async function everyRow<T>(name: string, params: Params = {}): Promise<T[]> {
  const rows: T[] = [];
  for (let page = 1; ; page += 1) {
    const got = await api.get<{ items: T[]; total: number }>(name, { ...params, page, limit: CEILING });
    rows.push(...got.items);
    if (got.items.length === 0 || rows.length >= got.total) return rows;
  }
}

/** The shoes on sale. The list and the options ask for the same thing, and `api.get` shares two identical requests made together. */
function liveShoes(): Promise<Product[]> {
  return everyRow<Product>("products", { status: "live" });
}

interface Names {
  line: Map<string, string>;
  category: Map<string, string>;
}

const NO_NAMES: Names = { line: new Map(), category: new Map() };

/** The names of the lines and categories, read only when words were typed: a word may be a line's name. */
async function namesFor(q: string): Promise<Names> {
  if (fold(q).trim() === "") return NO_NAMES;
  const [lines, categories] = await Promise.all([everyRow<Line>("lines"), everyRow<Category>("categories")]);
  return {
    line: new Map(lines.map((l) => [l.id, l.name])),
    category: new Map(categories.map((c) => [c.id, c.name])),
  };
}

/**
 * The answers the UI stage can come back with that are not a list: slow,
 * failed, or failed on page 2 only. Set by the browser's own storage
 * (`doi:mock` = shop-slow | shop-error | shop-more-error), never by a control
 * on the page; an offline browser fails like a 5xx. The real API's failures
 * replace all of this.
 */
function trouble(page: number): "slow" | "failed" | null {
  let flag: string | null = null;
  try {
    flag = window.localStorage.getItem("doi:mock");
  } catch {
    flag = null;
  }
  const offline = typeof navigator !== "undefined" && navigator.onLine === false;
  if (offline || flag === "shop-error" || (flag === "shop-more-error" && page > 1)) return "failed";
  return flag === "shop-slow" ? "slow" : null;
}

function inAny(list: readonly string[], value: string | undefined): boolean {
  return list.length === 0 || (value !== undefined && list.includes(value));
}

function inRanges(price: number, ranges: readonly PriceRange[]): boolean {
  return ranges.length === 0 || ranges.some((r) => (r.from === undefined || price >= r.from) && (r.to === undefined || price <= r.to));
}

function hasWords(p: Product, q: string, names: Names): boolean {
  const words = fold(q).split(/\s+/).filter(Boolean);
  if (words.length === 0) return true;
  const text = fold([p.name, p.model, p.colourway, p.code, names.line.get(p.line), names.category.get(p.category), p.description].join("\n"));
  return words.every((w) => text.includes(w));
}

function matches(p: Product, req: ProductsRequest, names: Names): boolean {
  const available = sizesInStock(p);
  return p.status === req.status
    && inAny(req.category, p.category)
    && inAny(req.line, p.line)
    && inAny(req.colour, p.colour)
    && inRanges(p.price, req.price)
    && (req.size.length === 0 || req.size.some((s) => available.includes(s)))
    && hasWords(p, req.q, names);
}

/** A shoe with no rating comes last on "Best rated". */
const ORDER: Record<Sort, (a: Product, b: Product) => number> = {
  "-released": (a, b) => (b.released ?? "").localeCompare(a.released ?? ""),
  price: (a, b) => a.price - b.price,
  "-price": (a, b) => b.price - a.price,
  "-rating": (a, b) => (b.rating ?? 0) - (a.rating ?? 0),
};

/** `GET /v1/products?status=live&limit=12&page=n&...` */
export async function listProducts(req: ProductsRequest): Promise<Page<Product>> {
  const bad = trouble(req.page);
  if (bad === "failed") throw new Error("The shoes could not be read");
  const [shoes, names] = await Promise.all([liveShoes(), namesFor(req.q)]);
  if (bad === "slow") await wait(3000);
  const hits = shoes.filter((p) => matches(p, req, names)).sort(ORDER[req.sort]);
  const start = (req.page - 1) * req.limit;
  return { items: hits.slice(start, start + req.limit), total: hits.length, page: req.page, limit: req.limit };
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
export async function getProductFilters(): Promise<ProductFilters> {
  const live = await liveShoes();
  const have = new Set(live.map((p) => p.colour));
  return {
    colours: colourOrder().filter((c) => have.has(c)),
    sizes: [...new Set(live.flatMap((p) => p.sizes ?? []))].sort((a, b) => sizeValue(a) - sizeValue(b)),
  };
}
