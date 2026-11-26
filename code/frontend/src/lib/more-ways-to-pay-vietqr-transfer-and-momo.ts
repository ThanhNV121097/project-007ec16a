import { useEffect, useState } from "react";
import { api, ApiError } from "./api";
import type { PaymentOption } from "./orders";
import type { OrderStatus } from "./shop";

/**
 * THE TWO NEW WAYS TO PAY, FROM THE BROWSER'S SIDE (docs/payments/stories/
 * more-ways-to-pay-vietqr-transfer-and-momo.md §3, §6): which ways the shop
 * lists, the bank-transfer details of an order, the dealer's "Mark as paid",
 * and MoMo's return address.
 *
 * It is the one file these screens reach the shop through, over src/lib/api.ts:
 * `GET /v1/payment-options`, `GET /v1/orders/{id}/transfer` with the order's
 * `X-Order-Token` (never in an address), and `PUT /v1/orders/{id}` whose body
 * carries `paid` and nothing else.
 *
 * No page holds a list of ways to pay: the ids come from the shop's own list
 * and from an order's `payment`. The two ids named here are the two ways that
 * behave differently from VNPay's — one has no payment page (its QR is drawn
 * on the order page), the other has its own words on the result page.
 */
export const BANK = "bank";
export const MOMO = "momo";

/** The ways the shop can complete right now, in the order it lists them. */
export async function readPaymentOptions(): Promise<PaymentOption[]> {
  const r = await api.get<{ items: PaymentOption[] }>("payment-options");
  return Array.isArray(r.items) ? r.items : [];
}

/** The ways the shop lists, those it refused to place an order with marked as not open. */
export function waysOf(listed: PaymentOption[], refused: string[]): PaymentOption[] {
  return listed.map((o) => (refused.includes(o.id) ? { ...o, is_available: false } : o));
}

/** The way the visitor picked while it is open, else the first open one; none when no way is open. */
export function chosenWay(ways: PaymentOption[], picked: string): PaymentOption | undefined {
  return ways.find((o) => o.id === picked && o.is_available) ?? ways.find((o) => o.is_available);
}

/** An unpaid, not cancelled bank-transfer order: it is paid by the QR code, never by "Pay now". */
export function payByTransfer(order: { payment: string; paid: boolean; status: string }): boolean {
  return order.payment === BANK && !order.paid && order.status !== "cancelled";
}

/** The shop's own account and what to type, and the QR code of the text it carries. */
export interface TransferDetails {
  holder: string;
  bank: string;
  account_number: string;
  /** In VND. */
  amount: number;
  /** The order's number. */
  note: string;
  payload: string;
  /** The code the shop drew: one string of 0 and 1 per row of modules, square, without the quiet zone. */
  qr: string[];
}

/** What the shop has said so far: nothing, no account any more, that it could not be asked, or the details. */
export type TransferRead =
  | { state: "loading" }
  | { state: "failed" }
  | { state: "gone" }
  | { state: "ready"; details: TransferDetails };

function detailsOf(body: unknown): TransferDetails | null {
  if (typeof body !== "object" || body === null) return null;
  const d = body as Record<string, unknown>;
  const text = ["holder", "bank", "account_number", "note", "payload"].every((k) => typeof d[k] === "string" && d[k] !== "");
  const qr = d.qr;
  const square =
    Array.isArray(qr) && qr.length >= 21 && qr.every((row) => typeof row === "string" && row.length === qr.length && /^[01]+$/.test(row));
  return text && typeof d.amount === "number" && square ? (d as unknown as TransferDetails) : null;
}

/**
 * The details of one order, asked with the order's token in `X-Order-Token` and
 * nowhere else (with none held the header is left out and a signed-in customer's
 * session speaks). A 503 (the account was removed) or a 409 (the order was paid,
 * cancelled or is no longer a bank transfer) means the block is not drawn; every
 * other failure, a 404 and a network failure included, is "could not load" — the
 * order page has already shown the order, so "not found" is never this call's.
 */
export async function readTransfer(id: string, token?: string): Promise<TransferRead> {
  try {
    const body = await api.get<unknown>(`orders/${encodeURIComponent(id)}/transfer`, undefined, token ? { "X-Order-Token": token } : undefined);
    const details = detailsOf(body);
    return details ? { state: "ready", details } : { state: "failed" };
  } catch (e) {
    return e instanceof ApiError && (e.status === 503 || e.status === 409) ? { state: "gone" } : { state: "failed" };
  }
}

/** The bank-transfer details of an order: `loading` from the first render of an order until the shop has answered for THAT order. */
export function useTransfer(id: string, token?: string): TransferRead {
  const [answered, setAnswered] = useState<{ id: string; read: TransferRead } | null>(null);
  useEffect(() => {
    let live = true;
    void readTransfer(id, token).then((read) => {
      if (live) setAnswered({ id, read });
    });
    return () => {
      live = false;
    };
  }, [id, token]);
  return answered && answered.id === id ? answered.read : { state: "loading" };
}

/** The shop's QR rows as dark (true) and light (false) modules. */
export function qrRows(qr: string[]): boolean[][] {
  return qr.map((row) => Array.from(row, (c) => c === "1"));
}

/** What "Mark as paid" came to: the order's status afterwards, a session that ended, or no answer. */
export type MarkResult = { ok: true; status: OrderStatus } | { ok: false; kind: "expired" | "failed" };

/** `PUT /v1/orders/{id}` with `{"paid": true}` and nothing else. */
export async function markPaid(id: string, status: OrderStatus): Promise<MarkResult> {
  try {
    const body = await api.put<{ status?: unknown } | undefined>(`orders/${encodeURIComponent(id)}`, { paid: true });
    const next = body?.status;
    return { ok: true, status: typeof next === "string" ? (next as OrderStatus) : status };
  } catch (e) {
    return { ok: false, kind: e instanceof ApiError && (e.status === 401 || e.status === 403) ? "expired" : "failed" };
  }
}

/**
 * What a payment provider's return address says: the order's reference and the
 * provider's result code. VNPay writes `vnp_TxnRef` and `vnp_ResponseCode` ("00"
 * is success); MoMo writes `orderId` and `resultCode` (0 is success), which is
 * given the page's one success word. An address of neither shape has neither.
 * Which provider the order uses is the shop's word, never this address's.
 */
export function readReturn(params: URLSearchParams): { reference: string | null; code: string | null } {
  const vnpay = params.get("vnp_TxnRef");
  if (vnpay !== null) return { reference: vnpay, code: params.get("vnp_ResponseCode") };
  const result = params.get("resultCode");
  return { reference: params.get("orderId"), code: result === "0" ? "00" : result };
}
