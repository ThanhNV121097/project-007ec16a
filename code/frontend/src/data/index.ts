/**
 * The site's records — shoes, lines, categories, reviews, orders, shop terms,
 * delivery, ways to pay, questions, the size chart — described in schema.json
 * and read from the API, a page at a time:
 *
 *   GET <base>/v1/<name>?page=&limit=&sort=&q=&<field>=<value> → { items, total, page, limit }
 *   GET <base>/v1/<name>/<id> → the record
 *
 * Components read records ONLY through these hooks. The base is
 * import.meta.env.VITE_API_URL ?? "/api" (src/lib/api.ts), and every request
 * runs in the browser, so a page paints first and its records arrive after:
 * `loading` and `error` say where a request is, and the design draws neither.
 *
 * `collections()` holds the rows loaded so far per name. The Studio and /admin
 * edit that browser copy with setCollection(): a name that has been set is
 * served from the copy, filtered here, and asks the server for nothing more.
 * Nothing is saved; a reload brings back what the database holds.
 */
import { useCallback, useEffect, useRef, useState, useSyncExternalStore } from "react";
import { api, ApiError, type Params } from "../lib/api";
import schemaFile from "./schema.json";

/** Where a record came from: the owner's own data, a page on the web, or an assumption. */
export type Source = "owner" | "assumed" | `web:${string}`;

export interface Row {
  id: string;
  _source?: Source;
  [field: string]: unknown;
}

export interface Field {
  key: string;
  label?: string;
  /** text | longtext | number | price | image | url | bool | ref:<collection> | list */
  type: string;
  currency?: string;
  /**
   * A closed set of values — a status, a size, a kind: ["draft", "live"], or
   * [["draft", "Draft"], ["live", "On sale"]] where the label differs. The
   * management area draws it as a list to choose from and a badge in a table,
   * instead of a box to type "live" into.
   */
  options?: readonly (string | readonly [string, string])[];
  /** A value only the design reads — a swatch token, a layout hint. Never
   *  shown on a management form: it is not the owner's content. */
  hidden?: boolean;
  /** Which tab of the record's page the field sits on. */
  section?: string;
  /** Saving is refused while this is empty. */
  required?: boolean;
  /** One line under the control, for what a label cannot say. */
  hint?: string;
}

export interface Collection {
  label?: string;
  fields: Field[];
}

export const schema = schemaFile as { collections: Record<string, Collection> };

/** The rows loaded so far per name, and the names whose copy was set by hand. */
let store: Record<string, Row[]> = {};
const edited = new Set<string>();
const listeners = new Set<() => void>();

/** Adds what an answer carried to the rows loaded so far, by id. */
function remember(name: string, rows: Row[]) {
  if (edited.has(name) || rows.length === 0) return;
  const known = store[name] ?? [];
  const fresh = new Map(rows.map((r) => [r.id, r]));
  const have = new Set(known.map((r) => r.id));
  store = { ...store, [name]: [...known.map((r) => fresh.get(r.id) ?? r), ...rows.filter((r) => !have.has(r.id))] };
}

/** The Studio's unsaved edits, shown live in its frame. */
export function setCollection(name: string, rows: Row[]) {
  edited.add(name);
  store = { ...store, [name]: rows };
  listeners.forEach((l) => l());
}

export function collections(): Record<string, Row[]> {
  return store;
}

function subscribe(l: () => void) {
  listeners.add(l);
  return () => {
    listeners.delete(l);
  };
}

/** The hand-edited copy of `name`, or undefined while the server is the source. */
function useEdited(name: string): Row[] | undefined {
  return useSyncExternalStore(subscribe, () => (edited.has(name) ? store[name] : undefined));
}

type Range = { from?: string | number; to?: string | number };

/**
 * One condition on a field. A plain value is an exact match; a list is "any
 * of" (a status filter with three boxes ticked); a range is from/to
 * inclusive, for dates and prices.
 *
 * An empty value — "", an empty list, a range with neither end — filters
 * nothing. A filter the person has not set yet must not empty the table.
 */
export type Filter = string | number | boolean | readonly (string | number)[] | Range;

export interface Query {
  /** Records whose field matches, e.g. { category: "iphone", status: ["pending", "rejected"] }. */
  where?: Record<string, Filter | undefined>;
  /** Free text, matched against `search` fields — accents and case ignored. */
  q?: string;
  /** Which fields `q` searches. Every text field when not given. */
  search?: readonly string[];
  /** "price" ascending, "-created_at" descending. */
  sort?: string;
  /**
   * The page to return, 1-based. Given, `items` is THAT page alone — the
   * shape a table needs. Left out, `items` grows with loadMore(), which is
   * what a catalogue's "Show more" needs.
   */
  page?: number;
  /** How many a page holds. */
  pageSize?: number;
}

function set(edge: string | number | undefined): boolean {
  return edge !== undefined && edge !== "";
}

function empty(v: Filter | undefined): boolean {
  if (v === undefined || v === null || v === "") return true;
  if (Array.isArray(v)) return v.length === 0;
  if (typeof v === "object") return !set((v as Range).from) && !set((v as Range).to);
  return false;
}

/** The query string a query stands for, without page and limit. */
function paramsOf(query: Query): Params {
  const out: Params = {};
  for (const [key, v] of Object.entries(query.where ?? {})) {
    if (empty(v)) continue;
    if (Array.isArray(v)) {
      out[key] = v.join(",");
    } else if (typeof v === "object") {
      const r = v as Range;
      if (set(r.from)) out[`${key}_from`] = r.from;
      if (set(r.to)) out[`${key}_to`] = r.to;
    } else {
      out[key] = v as string | number | boolean;
    }
  }
  if (query.q) out.q = query.q;
  if (query.sort) out.sort = query.sort;
  return out;
}

interface Answer {
  items: Row[];
  total: number;
  page: number;
  limit: number;
}

function fetchPage(name: string, params: Params, page: number, limit: number): Promise<Answer> {
  return api.get<Answer>(encodeURIComponent(name), { page, limit, ...params });
}

/** One page, and the limit the server applied (it serves at most 100). */
interface Loaded {
  items: Row[];
  total: number;
  limit: number;
  page: number;
}

function pagesOf(s: { total: number; limit: number }): number {
  return Math.max(1, Math.ceil(s.total / Math.max(1, s.limit)));
}

/**
 * The first page a query shows. A page past the last — the person stood on
 * page 7 while a filter narrowed the list — is asked about once more, for the
 * last page, instead of leaving an empty table with no way back.
 */
async function firstPage(name: string, params: Params, page: number, size: number): Promise<Loaded> {
  let r = await fetchPage(name, params, page, size);
  let at = page;
  if (page > 1 && r.items.length === 0 && r.total > 0) {
    at = pagesOf({ total: r.total, limit: r.limit || size });
    r = await fetchPage(name, params, at, size);
  }
  remember(name, r.items);
  return { items: r.items, total: r.total, limit: r.limit || size, page: at };
}

function asError(e: unknown): Error {
  return e instanceof Error ? e : new Error(String(e));
}

interface Remote extends Loaded {
  key: string;
  loading: boolean;
  error: Error | null;
}

interface Result {
  items: Row[];
  total: number;
  pages: number;
  page: number;
  hasMore: boolean;
  loading: boolean;
  error: Error | null;
  loadMore: () => void;
}

/** A collection read from the server: a request on mount, another per loadMore(). */
function useRemote(name: string, query: Query, off: boolean): Result {
  const size = query.pageSize ?? 12;
  const first = query.page === undefined ? 1 : Math.max(1, Math.floor(query.page));
  const params = paramsOf(query);
  const key = JSON.stringify([name, params, size, first]);
  const none: Remote = { key, items: [], total: 0, limit: size, page: first, loading: true, error: null };
  const [state, setState] = useState<Remote>({ ...none, key: "" });
  const latest = useRef({ key, state, name, params, size });
  latest.current = { key, state, name, params, size };
  const busy = useRef(false);

  useEffect(() => {
    if (off) return;
    let live = true;
    busy.current = false;
    const cur = latest.current;
    setState({ ...none, key });
    firstPage(cur.name, cur.params, first, cur.size).then(
      (r) => {
        if (live) setState({ ...r, key, loading: false, error: null });
      },
      (e) => {
        if (live) setState((s) => ({ ...s, loading: false, error: asError(e) }));
      },
    );
    return () => {
      live = false;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps -- the key stands for the query
  }, [key, off]);

  const loadMore = useCallback(() => {
    const cur = latest.current;
    const s = cur.state;
    if (busy.current || s.key !== cur.key || s.loading || s.page >= pagesOf(s)) return;
    busy.current = true;
    const next = s.page + 1;
    const mine = (p: Remote) => p.key === cur.key;
    setState((p) => (mine(p) ? { ...p, loading: true, error: null } : p));
    fetchPage(cur.name, cur.params, next, cur.size)
      .then(
        (r) => {
          remember(cur.name, r.items);
          setState((p) => (mine(p) ? { ...p, items: [...p.items, ...r.items], total: r.total, limit: r.limit || p.limit, page: next, loading: false } : p));
        },
        (e) => setState((p) => (mine(p) ? { ...p, loading: false, error: asError(e) } : p)),
      )
      .finally(() => {
        if (latest.current.key === cur.key) busy.current = false;
      });
  }, []);

  const s = state.key === key ? state : none;
  const pages = pagesOf(s);
  return { items: s.items, total: s.total, pages, page: s.page, hasMore: query.page === undefined && s.page < pages, loading: s.loading, error: s.error, loadMore };
}

/** Lowercased and stripped of accents, so "Cửa Lò" is found by "cua lo". */
function fold(v: unknown): string {
  return String(v ?? "").normalize("NFD").replace(/[\\u0300-\\u036f]/g, "").toLowerCase();
}

/**
 * value against one end of a range: negative below, positive above. Two
 * numbers compare as numbers; anything else as text, which is why a date is
 * kept as ISO — "2026-01-09" < "2026-02-01" is true read either way.
 */
function compare(value: unknown, edge: string | number): number {
  const x = Number(value);
  const y = Number(edge);
  if (!Number.isNaN(x) && !Number.isNaN(y) && edge !== "") return x - y;
  return String(value ?? "").localeCompare(String(edge));
}

function inRange(value: unknown, r: Range): boolean {
  if (set(r.from) && compare(value, r.from as string | number) < 0) return false;
  if (set(r.to) && compare(value, r.to as string | number) > 0) return false;
  return true;
}

function matches(row: Row, query: Query): boolean {
  const where = query.where ?? {};
  const ok = Object.entries(where).every(([k, v]) => {
    if (empty(v)) return true;
    const value = row[k];
    if (Array.isArray(v)) return v.some((one) => one === value || fold(one) === fold(value));
    if (typeof v === "object") return inRange(value, v as Range);
    return value === v || fold(v) === fold(value);
  });
  if (!ok) return false;
  const q = fold(query.q).trim();
  if (!q) return true;
  const fields = query.search ?? Object.keys(row).filter((k) => !k.startsWith("_") && typeof row[k] === "string");
  return fields.some((f) => fold(row[f]).includes(q));
}

/** Rows in `sort` order: "price" ascending, "-price" descending. */
function sorted(rows: Row[], sort?: string): Row[] {
  const key = (sort ?? "").trim();
  if (!key) return rows;
  const desc = key.startsWith("-");
  const field = desc ? key.slice(1) : key;
  const out = [...rows].sort((a, b) => {
    const x = a[field];
    const y = b[field];
    if (typeof x === "number" && typeof y === "number") return x - y;
    return String(x ?? "").localeCompare(String(y ?? ""), undefined, { numeric: true });
  });
  return desc ? out.reverse() : out;
}

/** A collection whose copy was set by hand: filtered, sorted and paged here. */
function useLocal(rows: Row[] | undefined, query: Query): Result {
  const size = query.pageSize ?? 12;
  const key = JSON.stringify([query.where ?? {}, query.q ?? "", query.search ?? null, query.sort ?? ""]);
  const [shown, setShown] = useState(1);
  useEffect(() => {
    setShown(1);
  }, [key]);
  const all = sorted((rows ?? []).filter((r) => matches(r, query)), query.sort);
  const pages = Math.max(1, Math.ceil(all.length / size));
  const page = query.page === undefined ? 1 : Math.min(Math.max(1, Math.floor(query.page)), pages);
  const items = query.page === undefined ? all.slice(0, shown * size) : all.slice((page - 1) * size, page * size);
  const loadMore = useCallback(() => setShown((p) => p + 1), []);
  return { items, total: all.length, pages, page, hasMore: query.page === undefined && items.length < all.length, loading: false, error: null, loadMore };
}

/**
 * A page at a time: `items` grows by `pageSize` with each `loadMore()` — wire
 * it to a "Show more" button or an IntersectionObserver for lazy loading.
 *
 * Returns { items, total, pages, page, hasMore, loading, error, loadMore }:
 * - `items` — the page asked for, or the records loaded so far;
 * - `total` — how many records match in all, loaded or not: the API's `total`,
 *   never `items.length`. A count the visitor reads ("24 products") is `total`;
 * - `pages` — how many pages of the limit the server applied those make;
 * - `page` — the page `items` holds: a page past the last shows the last page;
 * - `loading` — true while a request is out; `error` — the failure of the last
 *   request, with the items already loaded kept.
 *
 * Pass `page` for a dashboard list and leave it out for a catalogue, where
 * `loadMore()` appends the next page to the ones already shown. When the name
 * or the query changes, the old items are dropped and the first page of the
 * new query is asked for; an answer to an older query is discarded.
 */
// T is the record's shape as the page reads it. Any shape with an id will do —
// an interface included: `T extends Row` asked for Row's index signature,
// which no interface has, and a design turn spent 24 builds on the error.
export function useCollection<T extends { id: string } = Row>(name: string, query: Query = {}) {
  const edit = useEdited(name);
  const local = useLocal(edit, query);
  const remote = useRemote(name, query, edit !== undefined);
  const out = edit === undefined ? remote : local;
  return { ...out, items: out.items as unknown as T[] };
}

/** A record that is not there: the API's own `not_found`, not any 404 (a wrong address is a fault). */
function isNotFound(e: unknown): boolean {
  return e instanceof ApiError && e.status === 404 && e.code === "not_found";
}

interface One {
  key: string;
  item?: Row;
  loading: boolean;
  error: Error | null;
}

/** One record by id. A 404 is an answer: `item` is undefined and `error` null. */
export function useRecord<T extends { id: string } = Row>(name: string, id: string | undefined) {
  const edit = useEdited(name);
  const key = `${name}/${id ?? ""}`;
  const [state, setState] = useState<One>({ key: "", loading: true, error: null });
  const skip = edit !== undefined || id === undefined;
  useEffect(() => {
    if (skip || id === undefined) return;
    let live = true;
    setState({ key, loading: true, error: null });
    api.get<Row>(`${encodeURIComponent(name)}/${encodeURIComponent(id)}`).then(
      (item) => {
        remember(name, [item]);
        if (live) setState({ key, item, loading: false, error: null });
      },
      (e) => {
        if (live) setState({ key, loading: false, error: isNotFound(e) ? null : asError(e) });
      },
    );
    return () => {
      live = false;
    };
  }, [key, skip, name, id]);

  if (edit !== undefined) {
    return { item: edit.find((r) => r.id === id) as unknown as T | undefined, loading: false, error: null as Error | null };
  }
  const fresh = state.key === key;
  const waiting = id !== undefined && (!fresh || state.loading);
  return { item: (fresh ? state.item : undefined) as unknown as T | undefined, loading: waiting, error: fresh ? state.error : null };
}
