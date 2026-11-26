import { useCallback, useEffect, useState } from "react";
<<<<<<< refs/heads/feature/order-page-and-pay-now-be
import { api, ApiError } from "./api";
import { storedOrders } from "./orders";
import type { OrderStatus } from "./shop";

/**
 * THE ORDER PAGE'S CLIENT: the order read (GET /v1/orders/{id}) and the "Pay
 * now" call (POST /v1/orders/{id}/payment), the only file the page reaches the
 * shop through. Both go through src/lib/api.ts, with the cookie, and carry the
 * order's token as the `X-Order-Token` header when this browser holds one for
 * that id (read through the store in src/lib/orders.ts; none held: the same call
 * without the header, so a signed-in customer's session can open the order). The
 * token is never put in an address, and nothing here keeps one.
 *
 * The order is the shop's public view: never a phone, an email, an address or a
 * note.
=======
import type { OrderStatus } from "./shop";
import { getOrder, postPayment } from "./mock/order-page-and-pay-now";

/**
 * THE ORDER PAGE'S CLIENT: the order read and the "Pay now" call, the only
 * file the page reaches the shop through. Today it asks the mock; the backend
 * step makes both calls through src/lib/api.ts, with credentials, and reads
 * the order's token from the browser's store in src/lib/orders.ts for the
 * `X-Order-Token` header (no token held: the same call without it, so a
 * signed-in customer's session can open the order). The token is never put in
 * an address, and nothing here keeps one.
 *
 * The order is services.md §2.8's public view: never a phone, an email, an
 * address or a note.
>>>>>>> refs/heads/main
 */
export interface OrderItem {
  product: string;
  name: string;
  colourway: string;
  image: string;
  size: string;
  qty: number;
  /** As sold, in VND: not today's price. */
  unit_price: number;
}

export interface PublicOrder {
  id: string;
  number: string;
  /** RFC 3339, +07:00. */
  placed_at: string;
  status: OrderStatus;
  items: OrderItem[];
  /** The stored total, in VND. */
  total: number;
  /** Id of the way to pay and of the delivery option: names come from their records. */
  payment: string;
  paid: boolean;
  delivery: string;
  carrier: string;
  tracking: string;
  /** YYYY-MM-DD or "". */
  eta: string;
  customer: string;
  city: string;
}

/** What the shop has said so far: nothing, the order, "no such order", or that it could not be asked. */
export type OrderRead =
  | { state: "loading" }
  | { state: "missing" }
  | { state: "failed" }
  | { state: "ready"; order: PublicOrder };

<<<<<<< refs/heads/feature/order-page-and-pay-now-be
/** The header that opens this order, when this browser holds its token. */
function tokenHeader(id: string): Record<string, string> | undefined {
  const token = storedOrders().find((o) => o.id === id)?.token;
  return token ? { "X-Order-Token": token } : undefined;
}

function isOrder(body: unknown): body is PublicOrder {
  return typeof body === "object" && body !== null && Array.isArray((body as { items?: unknown }).items);
}

async function readOrder(id: string): Promise<OrderRead> {
  if (!id) return { state: "missing" };
  try {
    const body = await api.get<unknown>(`orders/${encodeURIComponent(id)}`, undefined, tokenHeader(id));
    return isOrder(body) ? { state: "ready", order: body } : { state: "failed" };
  } catch (e) {
    // Only the shop's own 404 is "not found": a 404 from a gateway, a 401, a 5xx and no
    // connection are all a failure to read, never a claim that the order does not exist.
    return e instanceof ApiError && e.status === 404 && e.code === "not_found" ? { state: "missing" } : { state: "failed" };
=======
async function readOrder(id: string): Promise<OrderRead> {
  try {
    const answer = await getOrder(id);
    if (answer.status === 200 && answer.body) return { state: "ready", order: answer.body as PublicOrder };
    // Only the shop's 404 is "not found": a 500, a 401 or a body with nothing in it is a failure to read.
    return answer.status === 404 ? { state: "missing" } : { state: "failed" };
  } catch {
    return { state: "failed" };
>>>>>>> refs/heads/main
  }
}

/**
 * The order at an address. `loading` from the first render of an id until the
 * shop has answered for THAT id — never another id's answer, never "missing"
 * before the answer — and again while `reload` asks a second time.
 */
export function useOrder(id: string | undefined): OrderRead & { reload: () => void } {
  const [attempt, setAttempt] = useState(0);
  const [answered, setAnswered] = useState<{ key: string; read: OrderRead } | null>(null);
  const key = `${id ?? ""}#${attempt}`;
  useEffect(() => {
    let current = true;
    readOrder(id ?? "").then((read) => {
      if (current) setAnswered({ key, read });
    });
    return () => {
      current = false;
    };
  }, [id, key]);
  const reload = useCallback(() => setAttempt((n) => n + 1), []);
  const read: OrderRead = answered && answered.key === key ? answered.read : { state: "loading" };
  return { ...read, reload };
}

/** "Pay now" is offered for an order that is unpaid and not cancelled. */
export function canPay(order: Pick<PublicOrder, "paid" | "status">): boolean {
  return !order.paid && order.status !== "cancelled";
}

/**
<<<<<<< refs/heads/feature/order-page-and-pay-now-be
 * One POST, no body, never tried again. Only a 200 answer carrying an http(s)
 * `payment_url` is a success; a 409, a 503, a 500, a 404, a network failure, an
 * answer with no address and an address of any other kind are all the same
 * failure, so the page needs no change when the shop starts answering with an
 * address.
 */
export async function requestPayment(id: string): Promise<{ ok: true; url: string } | { ok: false }> {
  try {
    const body = await api.post<{ payment_url?: unknown } | undefined>(`orders/${encodeURIComponent(id)}/payment`, undefined, tokenHeader(id));
    const url = body?.payment_url;
    return typeof url === "string" && /^https?:\/\//i.test(url) ? { ok: true, url } : { ok: false };
=======
 * One POST, no body. Only a 200 answer carrying an http(s) `payment_url` is a
 * success; a 409, a 503, a 500, a 404, a network failure, an answer with no
 * address and an address of any other kind are all the same failure, so the
 * page needs no change when the shop starts answering with an address.
 */
export async function requestPayment(id: string): Promise<{ ok: true; url: string } | { ok: false }> {
  try {
    const answer = await postPayment(id);
    const url = (answer.body as { payment_url?: unknown } | undefined)?.payment_url;
    return answer.status === 200 && typeof url === "string" && /^https?:\/\//i.test(url) ? { ok: true, url } : { ok: false };
>>>>>>> refs/heads/main
  } catch {
    return { ok: false };
  }
}

/** Sends the browser to the payment page. */
export function openPayment(url: string): void {
  window.location.assign(url);
}

/**
 * A time as Vietnam's clock reads it, as text with no zone — the form
 * formatDateTime reads as written — so the page shows 10:00 for
 * 2026-11-25T10:00:00+07:00 in every browser's zone.
 */
export function vietnamTime(value: string): string {
  const at = new Date(value);
  return Number.isNaN(at.getTime()) ? value : new Date(at.getTime() + 7 * 3600 * 1000).toISOString().slice(0, 19);
}
