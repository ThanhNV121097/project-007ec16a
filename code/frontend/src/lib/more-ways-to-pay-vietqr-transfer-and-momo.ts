import { useEffect, useState } from "react";
import { api } from "./api";
import type { PaymentOption } from "./orders";
import type { OrderStatus } from "./shop";
import {
  demoWay, getTransfer, mockOptions, putPaid, standInModules,
} from "./mock/more-ways-to-pay-vietqr-transfer-and-momo";

/**
 * THE TWO NEW WAYS TO PAY, FROM THE BROWSER'S SIDE (docs/payments/stories/
 * more-ways-to-pay-vietqr-transfer-and-momo.md §3, §6): which ways the shop
 * lists, the bank-transfer details of an order, the dealer's "Mark as paid",
 * and MoMo's return address.
 *
 * It is the one file these screens reach the shop through. Today the three
 * calls the shop does not answer yet come from the mock; the backend step makes
 * them `api.get("payment-options")`, `api.get("orders/{id}/transfer")` with the
 * order's `X-Order-Token`, and `api.put("orders/{id}", { paid: true })` — the
 * body carries `paid` and nothing else.
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
  const staged = mockOptions();
  if (staged) return staged;
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

/** The shop's own account and what to type, and the text the QR code carries. */
export interface TransferDetails {
  holder: string;
  bank: string;
  account_number: string;
  /** In VND. */
  amount: number;
  /** The order's number. */
  note: string;
  payload: string;
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
  return text && typeof d.amount === "number" ? (d as unknown as TransferDetails) : null;
}

/**
 * The details of one order. 503 is the account being gone (the block is not
 * drawn); every other failure, a 404 included, is "could not load" — the
 * order page has already shown the order, so "not found" is never this call's.
 */
export async function readTransfer(number: string, total: number): Promise<TransferRead> {
  try {
    const answer = await getTransfer(number, total);
    if (answer.status === 503) return { state: "gone" };
    const details = answer.status === 200 ? detailsOf(answer.body) : null;
    return details ? { state: "ready", details } : { state: "failed" };
  } catch {
    return { state: "failed" };
  }
}

/** The bank-transfer details of an order: `loading` from the first render of an order until the shop has answered for THAT order. */
export function useTransfer(order: { id: string; number: string; total: number }): TransferRead {
  const [answered, setAnswered] = useState<{ id: string; read: TransferRead } | null>(null);
  useEffect(() => {
    let live = true;
    void readTransfer(order.number, order.total).then((read) => {
      if (live) setAnswered({ id: order.id, read });
    });
    return () => {
      live = false;
    };
  }, [order.id, order.number, order.total]);
  return answered && answered.id === order.id ? answered.read : { state: "loading" };
}

/** The picture of a QR payload, as rows of dark and light modules. */
export function qrModules(payload: string): boolean[][] {
  return standInModules(payload);
}

/** What "Mark as paid" came to: the order's status afterwards, a session that ended, or no answer. */
export type MarkResult = { ok: true; status: OrderStatus } | { ok: false; kind: "expired" | "failed" };

export async function markPaid(id: string, status: OrderStatus): Promise<MarkResult> {
  try {
    const answer = await putPaid(id, status);
    if (answer.status === 200) {
      const next = (answer.body as { status?: unknown } | undefined)?.status;
      return { ok: true, status: typeof next === "string" ? (next as OrderStatus) : status };
    }
    return { ok: false, kind: answer.status === 401 || answer.status === 403 ? "expired" : "failed" };
  } catch {
    return { ok: false, kind: "failed" };
  }
}

/** The order page's order as the address asks to see it (`?mock=bank`); the shop's own order when it asks for none. */
export const viaDemo = demoWay;

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
