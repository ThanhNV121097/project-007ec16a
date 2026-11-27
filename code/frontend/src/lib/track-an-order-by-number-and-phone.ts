import { ApiError, api } from "./api";
import { keepOrder, storedOrders, type StoredOrder } from "./orders";

/**
 * LOOKING AN ORDER UP AGAIN, FROM THE BROWSER'S SIDE (POST /v1/order-lookups,
 * docs/architecture/services.md §3.12): the one call, the reading of its
 * answers, and the list of orders this browser already holds.
 *
 * The page sends what was typed, as typed: the shop reads the digits, and it
 * is the shop that says whether the pair matches. The answer is an id, a
 * number, a time and a token and nothing else; the phone is never kept, never
 * put in an address and never logged. The call is never retried (one request
 * per press), and an order is saved through the same store checkout writes.
 */
export interface LookupBody {
  number: string;
  phone: string;
}

/** What the shop answers to a right pair. */
export interface LookupAnswer {
  id: string;
  number: string;
  placed_at: string;
  token: string;
}

export type LookupFailure = "not_found" | "too_many" | "offline" | "failed";

export type LookupResult = { ok: true; order: StoredOrder } | { ok: false; kind: LookupFailure };

const offline = () => typeof navigator !== "undefined" && navigator.onLine === false;

/** The transport: POST ${base}/v1/order-lookups on the project's one client, never retried. */
const send = (body: LookupBody): Promise<unknown> => api.post<LookupAnswer>("order-lookups", body);

const ID = /^[a-z0-9][a-z0-9._-]*$/;
const isId = (id: unknown): id is string => typeof id === "string" && id.length <= 100 && ID.test(id);

/** An answer that is the four things a right pair gives, and the four only. */
function readAnswer(answer: unknown): StoredOrder | null {
  if (typeof answer !== "object" || answer === null) return null;
  const { id, number, placed_at, token } = answer as Record<string, unknown>;
  if (!isId(id) || typeof number !== "string" || typeof placed_at !== "string") return null;
  if (typeof token !== "string" || token === "") return null;
  return { id, number, placed_at, token };
}

/**
 * Which sentence a failure is told in. A 429 is read from its status and a
 * not-found only from the shop's own `not_found` code: a gateway's own 404
 * page is "could not look that up", never "no order matches".
 */
function failureOf(e: unknown): LookupFailure {
  if (!(e instanceof ApiError)) return offline() ? "offline" : "failed";
  if (e.status === 429) return "too_many";
  if (e.status === 404 && e.code === "not_found") return "not_found";
  if (e.status === 0 && offline()) return "offline";
  return "failed";
}

/** Looks the order up and, on a right pair, saves it in this browser. Never throws. */
export async function lookupOrder(number: string, phone: string): Promise<LookupResult> {
  if (offline()) return { ok: false, kind: "offline" };
  try {
    const order = readAnswer(await send({ number, phone }));
    if (!order) return { ok: false, kind: "failed" };
    keepOrder(order);
    return { ok: true, order };
  } catch (e) {
    return { ok: false, kind: failureOf(e) };
  }
}

const SHOWN = 5;

function isHeld(o: unknown): o is StoredOrder {
  if (typeof o !== "object" || o === null) return false;
  const { id, number, placed_at } = o as Record<string, unknown>;
  return isId(id) && typeof number === "string" && typeof placed_at === "string";
}

function placedAt(o: StoredOrder): number {
  const t = Date.parse(o.placed_at);
  return Number.isNaN(t) ? 0 : t;
}

/** The orders this browser holds a token for: the five placed last, newest first. No request is made. */
export function heldOrders(): StoredOrder[] {
  return storedOrders()
    .filter(isHeld)
    .sort((a, b) => placedAt(b) - placedAt(a))
    .slice(0, SHOWN);
}
