/**
 * THE SHOP'S WRITE ENDPOINTS, STANDING IN FOR A BACKEND THAT DOES NOT EXIST YET.
 *
 * Deleted by the backend step, which points the transport in
 * src/lib/manage-shoes-lines-categories-reviews-and-orders.ts at api.ts. It
 * keeps its pretend records in the same browser copy the Studio edits
 * (setCollection) and never touches browser storage: a reload brings back what
 * the database holds.
 *
 * HOW A REVIEWER REACHES EACH ANSWER — by what is typed, never by a control:
 *   a name containing "slow"      the shop takes 2 seconds
 *   a name containing "offline"   no answer     "expired"  401     "broken"  500
 *   no name, a price below 0, a count below 0, a star count outside 1-5   422
 *   a count of 13 on a size       409: an order "took 2 pairs" since it was read
 *   a count of 14 on a size       no answer
 *   a picture: GIF 415, over 5 MB 413, a file name with "offline" no answer
 *   Delete: samba-og-w-core-black 409 (an order refers to it); a line or
 *   category that loaded shoes use 409
 */
import { collections, schema, setCollection, type Row } from "../../data";
import type { Body, SizeChange, Transport, Wire } from "../manage-shoes-lines-categories-reviews-and-orders";

const sleep = (ms: number) => new Promise<void>((done) => setTimeout(done, ms));
const TAKEN = "Pairs were just taken by an order. Check the number and save again.";

function say(status: number, error: string, message: string, errors?: Record<string, string>): Wire {
  return { status, body: { error, message, ...(errors ? { errors } : {}) } };
}

/** The typed text asks the pretend shop to be slow or to fail. */
async function behave(text: string): Promise<Wire | null> {
  const t = text.toLowerCase();
  if (t.includes("slow")) await sleep(2000);
  if (t.includes("offline")) return { status: 0 };
  if (t.includes("expired")) return say(401, "unauthenticated", "Sign in to continue.");
  if (t.includes("broken")) return say(500, "internal", "Something went wrong.");
  return null;
}

const blank = (v: unknown) => v === undefined || v === null || v === "" || (Array.isArray(v) && v.length === 0);
const rowsOf = (collection: string): Row[] => collections()[collection] ?? [];
const keep = (collection: string, rows: Row[]) => setCollection(collection, rows);

type Shelf = Record<string, number>;
const shelves = new Map<string, Shelf>();

/** The counts of a shoe: those held, those it came with, or a guess from its lists. */
function shelfOf(row: Row): Shelf {
  const held = shelves.get(row.id);
  if (held) return held;
  if (row.pairs && typeof row.pairs === "object") return { ...(row.pairs as Shelf) };
  const list = (k: string): string[] => (Array.isArray(row[k]) ? (row[k] as unknown[]).map(String) : []);
  const out: Shelf = {};
  for (const s of list("sizes")) out[s] = list("sizes_out").includes(s) ? 0 : list("sizes_low").includes(s) ? 1 : 3;
  return out;
}

/** The shoe with its lists and total derived from its counts, as the shop reads them. */
function withShelf(row: Row, shelf: Shelf): Row {
  shelves.set(row.id, shelf);
  const sizes = Object.keys(shelf);
  return {
    ...row, pairs: { ...shelf }, sizes,
    sizes_low: sizes.filter((s) => shelf[s] > 0 && shelf[s] <= 2),
    sizes_out: sizes.filter((s) => shelf[s] === 0),
    stock: sizes.reduce((sum, s) => sum + shelf[s], 0),
  };
}

function fieldRefusals(collection: string, body: Body, partial: boolean): Record<string, string> {
  const out: Record<string, string> = {};
  const fields = schema.collections[collection]?.fields ?? [];
  const known = new Set(fields.map((f) => f.key));
  for (const key of Object.keys(body)) if (!known.has(key) && key !== "pairs" && key !== "seen_pairs") out[key] = "This field is not part of the record";
  for (const f of fields) {
    const v = body[f.key];
    if (f.required && (!partial || f.key in body) && blank(v)) out[f.key] = `${f.label ?? f.key} is needed`;
    if (blank(v)) continue;
    if ((f.type === "price" || f.type === "number") && !Number.isInteger(Number(v))) out[f.key] = "Enter a whole number";
    else if (f.type === "price" && Number(v) < 0) out[f.key] = "Price cannot be below 0";
    else if (collection === "reviews" && f.key === "rating" && (Number(v) < 1 || Number(v) > 5)) out[f.key] = "Stars must be 1 to 5";
    if (f.options && !f.options.some((o) => (typeof o === "string" ? o : o[0]) === v)) out[f.key] = "Choose one of the choices";
  }
  return out;
}

function pairRefusals(pairs: unknown): Record<string, string> {
  const out: Record<string, string> = {};
  for (const [size, n] of Object.entries((pairs ?? {}) as Record<string, unknown>)) {
    if (!Number.isInteger(n) || (n as number) < 0) out[`pairs.${size}`] = "Use a whole number, 0 or more";
  }
  return out;
}

function slug(name: string): string {
  const plain = name.normalize("NFD").replace(/[̀-ͯ]/g, "").replace(/đ/gi, "d");
  return plain.toLowerCase().replace(/[^a-z0-9]+/g, "-").replace(/^-|-$/g, "");
}

function idFor(collection: string, name: string, rows: Row[]): string {
  if (collection === "reviews") return `r-${Math.random().toString(16).slice(2, 10).padEnd(8, "0")}`;
  const base = slug(name) || collection;
  let id = base;
  for (let n = 2; rows.some((r) => r.id === id); n += 1) id = `${base}-${n}`;
  return id;
}

const trimmed = (body: Body): Body => Object.fromEntries(Object.entries(body).map(([k, v]) => [k, typeof v === "string" ? v.trim() : v]));

/** One shoe opens the home page: saving another with it on turns it off here. */
function onlyFeatured(rows: Row[], saved: Row): Row[] {
  return saved.featured === true ? rows.map((r) => (r.id === saved.id ? r : { ...r, featured: false })) : rows;
}

async function create(collection: string, body: Body): Promise<Wire> {
  const stop = await behave(String(body.name ?? ""));
  if (stop) return stop;
  const bad = { ...fieldRefusals(collection, body, false), ...pairRefusals(body.pairs) };
  if (Object.keys(bad).length > 0) return say(422, "invalid", "Some fields need another look.", bad);
  const { pairs, seen_pairs: _seen, ...rest } = trimmed(body);
  const rows = rowsOf(collection);
  let saved: Row = { ...rest, id: idFor(collection, String(rest.name ?? ""), rows) };
  if (collection === "products") saved = withShelf({ rating: 0, review_count: 0, ...saved }, (pairs ?? {}) as Shelf);
  keep(collection, [...onlyFeatured(rows, saved), saved]);
  return { status: 201, body: saved };
}

async function update(collection: string, id: string, body: Body): Promise<Wire> {
  const rows = rowsOf(collection);
  const row = rows.find((r) => r.id === id);
  if (!row) return say(404, "not_found", "Not found.");
  const stop = await behave(String(body.name ?? row.name ?? ""));
  if (stop) return stop;
  const bad = { ...fieldRefusals(collection, body, true), ...pairRefusals(body.pairs) };
  if (Object.keys(bad).length > 0) return say(422, "invalid", "Some fields need another look.", bad);
  const taken = Object.entries((body.pairs ?? {}) as Shelf).filter(([, n]) => n === 13);
  if (taken.length > 0) return say(409, "stock_conflict", TAKEN, Object.fromEntries(taken.map(([s]) => [`pairs.${s}`, TAKEN])));
  const { pairs, seen_pairs: _seen, ...rest } = trimmed(body);
  let saved: Row = { ...row, ...rest, id };
  if (collection === "products") saved = withShelf(saved, pairs ? (pairs as Shelf) : shelfOf(row));
  keep(collection, onlyFeatured(rows.map((r) => (r.id === id ? saved : r)), saved));
  return { status: 200, body: saved };
}

async function remove(collection: string, id: string): Promise<Wire> {
  const rows = rowsOf(collection);
  const row = rows.find((r) => r.id === id);
  if (!row) return say(404, "not_found", "Not found.");
  const stop = await behave(String(row.name ?? ""));
  if (stop) return stop;
  const ordered = rowsOf("orders").some((o) => o.product === id);
  if (collection === "products" && (ordered || id === "samba-og-w-core-black")) {
    return say(409, "conflict", "Orders refer to this shoe. Set it to draft instead.");
  }
  const column = collection === "lines" ? "line" : collection === "categories" ? "category" : "";
  const using = column ? rowsOf("products").filter((p) => p[column] === id).length : 0;
  if (using > 0) return say(409, "conflict", `${using} shoes use this. Move them first.`);
  keep(collection, rows.filter((r) => r.id !== id));
  return { status: 204 };
}

async function read(collection: string, id: string): Promise<Wire> {
  const row = rowsOf(collection).find((r) => r.id === id);
  if (!row) return say(404, "not_found", "Not found.");
  return { status: 200, body: collection === "products" ? withShelf(row, shelfOf(row)) : row };
}

async function upload(file: File): Promise<Wire> {
  await sleep(2000);
  if (file.name.toLowerCase().includes("offline")) return { status: 0 };
  if (!["image/jpeg", "image/png", "image/webp"].includes(file.type)) return say(415, "unsupported_media_type", "Choose a JPEG, PNG or WebP picture.");
  if (file.size > 5_242_880) return say(413, "payload_too_large", "That picture is too large (5 MB at most).");
  const url = typeof URL.createObjectURL === "function" ? URL.createObjectURL(file) : `/api/v1/uploads/${slug(file.name)}.png`;
  return { status: 201, body: { url } };
}

async function setPairs(id: string, change: SizeChange): Promise<Wire> {
  await sleep(1000);
  if (change.pairs === 14) return { status: 0 };
  const row = rowsOf("products").find((r) => r.id === id);
  if (!row) return say(404, "not_found", "Not found.");
  const shelf = shelfOf(row);
  if (!(change.size in shelf)) return say(404, "not_found", "That size is not listed.");
  const lost = change.pairs === 13;
  const next = withShelf(row, { ...shelf, [change.size]: lost ? Math.max(0, change.seen_pairs - 2) : change.pairs });
  keep("products", rowsOf("products").map((r) => (r.id === id ? next : r)));
  if (lost) return say(409, "conflict", TAKEN);
  const { sizes, sizes_low, sizes_out, stock } = next;
  return { status: 200, body: { size: change.size, pairs: change.pairs, sizes, sizes_low, sizes_out, stock } };
}

const transport: Transport = { create, update, remove, read, upload, setPairs };
export default transport;
