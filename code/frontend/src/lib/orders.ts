import { useEffect, useState } from "react";
import { api, apiBase } from "./api";
import type { CartLine } from "./cart";
import { digitsOf } from "./shop";

/**
 * PLACING AN ORDER, FROM THE BROWSER'S SIDE (docs/architecture/services/
 * checkout-and-placing-an-order.md §3.1, §3.4): the request the shop is sent,
 * the Idempotency-Key that makes a repeated send safe, what each answer means
 * to the page, and the store of orders this browser has placed.
 *
 * This is the one file Checkout reads orders through. A POST is never retried
 * (a second send would be a second order unless the key says otherwise), so it
 * does not go through api.get; it uses the same base and the same envelope.
 *
 * The request never carries a price, a delivery fee or a total: the shop reads
 * its own prices and the browser's numbers are not trusted.
 */
export interface OrderRequest {
  items: { product: string; size: string; qty: number }[];
  delivery: "standard";
  payment: "vnpay";
  customer: string;
  phone: string;
  address: string;
  city: string;
  note: string;
}

/** What the transport hands back: the HTTP status and the JSON body. */
export interface Wire {
  status: number;
  body: Record<string, unknown>;
}

export interface PaymentOption {
  id: string;
  name: string;
  note: string;
  icon: string;
  online: boolean;
  is_available: boolean;
}

export interface PlacedOrder {
  id: string;
  number: string;
  placed_at: string;
  status: string;
  paid: boolean;
  total: number;
}

export interface Placed {
  order: PlacedOrder;
  token: string;
  /** Where to pay; "" when the shop has no address to send the browser to. */
  payment_url: string;
}

export interface Details {
  name: string;
  phone: string;
  address: string;
  city: string;
  note: string;
}

/** What the visitor typed that the shop would refuse, as the page words it. */
export function detailProblems(d: Details): ("missing" | "phone")[] {
  const out: ("missing" | "phone")[] = [];
  if (!d.name.trim() || !d.address.trim() || !d.city.trim()) out.push("missing");
  if (digitsOf(d.phone).length < 9) out.push("phone");
  return out;
}

/** A bag line's variant ("EU 36 2/3") as the size the shoe lists ("36 2/3"). */
export function sizeOfVariant(variant?: string): string {
  return (variant ?? "").replace(/^EU\s+/i, "").trim();
}

export function buildRequest(lines: CartLine[], d: Details): OrderRequest {
  return {
    items: lines.map((l) => ({ product: l.id, size: sizeOfVariant(l.variant), qty: l.qty })),
    delivery: "standard",
    payment: "vnpay",
    customer: d.name.trim(),
    phone: d.phone.trim(),
    address: d.address.trim(),
    city: d.city.trim(),
    note: d.note.trim(),
  };
}

function newKey(): string {
  const c = globalThis.crypto;
  if (c && typeof c.randomUUID === "function") return c.randomUUID();
  return `k-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 12)}`;
}

/**
 * One Idempotency-Key for one bag with one set of details: asking again with
 * the same request gives the same key (so a send whose answer was lost lands
 * on the order it made), and any change to the bag or a detail gives a new one.
 */
export function keyKeeper(): (request: OrderRequest) => string {
  let seen = "";
  let key = "";
  return (request) => {
    const now = JSON.stringify(request);
    if (key === "" || now !== seen) {
      seen = now;
      key = newKey();
    }
    return key;
  };
}

export type FailureKind = "refused" | "invalid" | "unavailable" | "offline" | "failed";

export type PlaceResult =
  | { ok: true; placed: Placed }
  | { ok: false; kind: FailureKind; problems: string[] };

/** The sentences of an answer's `errors`, in the order of the pairs they name. */
function sentences(body: Wire["body"]): string[] {
  const errors = body.errors;
  if (!errors || typeof errors !== "object") return [];
  return Object.entries(errors as Record<string, unknown>)
    .filter((e): e is [string, string] => typeof e[1] === "string")
    .sort(([a], [b]) => a.localeCompare(b, undefined, { numeric: true }))
    .map(([, sentence]) => sentence);
}

function placedOf(body: Wire["body"]): Placed | null {
  const order = body.order as Partial<PlacedOrder> | undefined;
  if (!order || typeof order.id !== "string" || typeof body.token !== "string") return null;
  return { order: order as PlacedOrder, token: body.token, payment_url: typeof body.payment_url === "string" ? body.payment_url : "" };
}

function interpret({ status, body }: Wire): PlaceResult {
  const placed = status === 200 || status === 201 ? placedOf(body) : null;
  if (placed) return { ok: true, placed };
  if (status === 409 && body.error === "stock_conflict") return { ok: false, kind: "refused", problems: sentences(body) };
  if (status === 422) return { ok: false, kind: "invalid", problems: sentences(body) };
  if (status === 503 && body.error === "payment_unavailable") return { ok: false, kind: "unavailable", problems: [] };
  return { ok: false, kind: "failed", problems: [] };
}

const offline = () => typeof navigator !== "undefined" && navigator.onLine === false;

const TIMEOUT_MS = 15_000;

/** POST /v1/orders with the key as a header. One try, abandoned after 15 seconds; never retried. */
async function postOrder(request: OrderRequest, key: string): Promise<Wire> {
  const abort = new AbortController();
  const timer = setTimeout(() => abort.abort(), TIMEOUT_MS);
  try {
    const res = await fetch(`${apiBase()}/v1/orders`, {
      method: "POST",
      credentials: "include",
      signal: abort.signal,
      headers: { Accept: "application/json", "Content-Type": "application/json", "Idempotency-Key": key },
      body: JSON.stringify(request),
    });
    const body: unknown = await res.json().catch(() => ({}));
    return { status: res.status, body: typeof body === "object" && body !== null ? (body as Record<string, unknown>) : {} };
  } finally {
    clearTimeout(timer);
  }
}

/** Sends the order. Never throws: every way it can go wrong is a `PlaceResult`. */
export async function placeOrder(request: OrderRequest, key: string): Promise<PlaceResult> {
  if (offline()) return { ok: false, kind: "offline", problems: [] };
  try {
    return interpret(await postOrder(request, key));
  } catch {
    return { ok: false, kind: offline() ? "offline" : "failed", problems: [] };
  }
}

/** The address to send the browser to, only when it is an https: address. */
export function payAddress(url: string): string | null {
  try {
    return new URL(url).protocol === "https:" ? url : null;
  } catch {
    return null;
  }
}

/**
 * The ways to pay the shop offers (GET /v1/payment-options): `loading` until the
 * read has finished, and then whatever it gave — an empty list when it failed,
 * which Checkout draws as the unavailable state.
 */
export function usePaymentOptions(): { items: PaymentOption[]; loading: boolean } {
  const [state, setState] = useState<{ items: PaymentOption[]; loading: boolean }>({ items: [], loading: true });
  useEffect(() => {
    let live = true;
    api
      .get<{ items: PaymentOption[] }>("payment-options")
      .then((r) => live && setState({ items: Array.isArray(r.items) ? r.items : [], loading: false }))
      .catch(() => live && setState({ items: [], loading: false }));
    return () => {
      live = false;
    };
  }, []);
  return state;
}

/**
 * The shop's price today for each shoe in the bag (GET /v1/products/{id}, one per
 * distinct shoe — a bag holds a few). A shoe the shop does not answer for keeps
 * the price its bag line saved; the shop's answer to placing is the authority.
 */
export function useCurrentPrices(ids: string[]): Map<string, number> {
  const wanted = [...new Set(ids)].sort().join(",");
  const [prices, setPrices] = useState<Map<string, number>>(new Map());
  useEffect(() => {
    let live = true;
    for (const id of wanted.split(",").filter(Boolean)) {
      api
        .get<{ price?: number }>(`products/${encodeURIComponent(id)}`)
        .then((p) => {
          if (live && typeof p.price === "number") setPrices((m) => new Map(m).set(id, p.price as number));
        })
        .catch(() => undefined);
    }
    return () => {
      live = false;
    };
  }, [wanted]);
  return prices;
}

/**
 * The orders this browser has placed, kept under a key of its own — never
 * the design's "doi:orders" — as the order's id, number, placed time and
 * token and nothing else about it. A browser that refuses storage keeps them
 * for this page load.
 */
export interface StoredOrder {
  id: string;
  number: string;
  placed_at: string;
  token: string;
}

const STORE = "doi:placed";
let memory: StoredOrder[] = [];

function onDisk(): StoredOrder[] {
  try {
    const v = JSON.parse(localStorage.getItem(STORE) ?? "null") as unknown;
    return Array.isArray(v) ? (v as StoredOrder[]) : [];
  } catch {
    return [];
  }
}

/** Newest first: what storage holds, and what only this page load remembers. */
export function storedOrders(): StoredOrder[] {
  const disk = onDisk();
  return [...memory.filter((m) => !disk.some((d) => d.id === m.id)), ...disk];
}

export function rememberOrder(placed: Placed): void {
  const { id, number, placed_at } = placed.order;
  keepOrder({ id, number, placed_at, token: placed.token });
}

/**
 * The one writer of the store: an order placed at checkout and an order found
 * again on /track both come through here, so there is one key and one shape.
 * An order already held is replaced, so it stays one entry with the newest token.
 */
export function keepOrder(kept: StoredOrder): void {
  memory = [kept, ...memory.filter((o) => o.id !== kept.id)];
  try {
    localStorage.setItem(STORE, JSON.stringify([kept, ...onDisk().filter((o) => o.id !== kept.id)]));
  } catch {
    /* storage refused: the order is placed and the token lasts this page load */
  }
}
