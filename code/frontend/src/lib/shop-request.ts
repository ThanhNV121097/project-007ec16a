import { PRICE_BANDS } from "./shop";
import type { TableQuery } from "./table";

/**
 * WHAT THE SHOP ASKS THE API FOR, written once: the address bar read into a
 * clean query, and that query turned into the request the product list
 * answers. Nothing here knows where the answer comes from.
 *
 * The shop asks for twelve shoes at a time, never more than the API's 100.
 * `size` in the address is an EU size on this page, never a page size, so the
 * limit is this constant and not the hook's `size`.
 */
export const PAGE_SIZE = 12;

export const SORTS = ["-released", "price", "-price", "-rating"] as const;
export type Sort = (typeof SORTS)[number];
export const DEFAULT_SORT: Sort = "-released";

/** The address bar, cleaned: only price bands the shop has, only sorts it offers. */
export interface ShopQuery {
  category: string[];
  line: string[];
  colour: string[];
  /** EU sizes the pair must have available. */
  size: string[];
  /** Price band ids; one or several, any of them. */
  price: string[];
  q: string;
  sort: Sort;
}

/** One inclusive range of prices; an end left out is open. */
export interface PriceRange {
  from?: number;
  to?: number;
}

/**
 * One page of live shoes. A union of price bands is a list of ranges here;
 * how a request carries it over the wire is the technical design's decision.
 */
export interface ProductsRequest {
  status: "live";
  page: number;
  limit: number;
  q: string;
  sort: Sort;
  category: string[];
  line: string[];
  colour: string[];
  size: string[];
  price: PriceRange[];
}

/** The list answer: `total` is every match, not the length of `items`. */
export interface Page<T> {
  items: T[];
  total: number;
  page: number;
  limit: number;
}

/** `GET /v1/product-filters`: every colour and EU size a live shoe has. */
export interface ProductFilters {
  colours: string[];
  sizes: string[];
}

function isSort(value: string): value is Sort {
  return (SORTS as readonly string[]).includes(value);
}

/** A band id the shop knows; `?price=abc` is as if no price were set. */
function isBand(id: string): boolean {
  return PRICE_BANDS.some((b) => b.id === id);
}

export function readShopQuery(t: Pick<TableQuery, "list" | "q" | "sort">): ShopQuery {
  return {
    category: t.list("category"),
    line: t.list("line"),
    colour: t.list("colour"),
    size: t.list("size"),
    price: t.list("price").filter(isBand),
    q: t.q.trim(),
    sort: isSort(t.sort) ? t.sort : DEFAULT_SORT,
  };
}

/** The ranges of the ticked bands: ticking two shows shoes in either. */
export function bandRanges(ids: readonly string[]): PriceRange[] {
  return PRICE_BANDS.filter((b) => ids.includes(b.id)).map((b) => ({
    ...(b.from > 0 ? { from: b.from } : {}),
    ...(b.to < Number.MAX_SAFE_INTEGER ? { to: b.to } : {}),
  }));
}

export function toRequest(query: ShopQuery, page: number): ProductsRequest {
  return {
    status: "live",
    page,
    limit: PAGE_SIZE,
    q: query.q,
    sort: query.sort,
    category: query.category,
    line: query.line,
    colour: query.colour,
    size: query.size,
    price: bandRanges(query.price),
  };
}
