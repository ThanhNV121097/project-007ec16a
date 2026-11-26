import { useCallback, useEffect, useMemo, useState } from "react";
import { api, ApiError } from "./api";
import { payAddress, storedOrders, type StoredOrder } from "./orders";

/**
 * THE PAYMENT RESULT PAGE, FROM THE BROWSER'S SIDE (docs/payments/stories/
 * online-payment-with-vnpay.md §3, §5): VNPay sends the customer to
 * /payment/return?vnp_TxnRef=<order number>-<n>&vnp_ResponseCode=<code>…, and
 * this file turns that address into what the page says.
 *
 * The address is NEVER believed about money. The reference only picks which
 * stored token to ask the shop with (through the token store's own reader in
 * ./orders — no second store); the response code only chooses between
 * "confirming" and "not completed" for an order the shop still shows unpaid.
 * That an order is paid, or cancelled, is the shop's word alone, and nothing
 * here marks anything paid.
 *
 * The calls go through one `transport`: the shop's GET /v1/orders/{id} and
 * POST /v1/orders/{id}/payment, each with the order's token in X-Order-Token and
 * nowhere else (never an address, a query or a log).
 */
export type PaymentState = "paid" | "confirming" | "timed-out" | "not-completed" | "cancelled" | "no-match" | "check-failed";

/** What the shop's answer says of an order: the two facts the page needs. */
export interface OrderState {
  id: string;
  number: string;
  status: string;
  paid: boolean;
}

/** What the transport hands back: the HTTP status (0 when no answer came) and the JSON body. */
export interface Wire {
  status: number;
  body: unknown;
}

export interface Transport {
  read(id: string, token: string): Promise<Wire>;
  pay(id: string, token: string): Promise<Wire>;
}

/**
 * What the shop's answer is on the wire: a 2xx is status 200 with its body, a
 * refusal is its status with `{ error: <code> }`, and no answer at all (no
 * connection, a timeout, anything thrown) is status 0.
 */
async function wire(run: () => Promise<unknown>): Promise<Wire> {
  try {
    return { status: 200, body: await run() };
  } catch (e) {
    if (e instanceof ApiError && e.status !== 0) return { status: e.status, body: { error: e.code } };
    return { status: 0, body: undefined };
  }
}

const transport: Transport = {
  read: (id, token) => wire(() => api.get<unknown>(`orders/${encodeURIComponent(id)}`, undefined, { "X-Order-Token": token })),
  // No body, sent once, never retried: a press is not idempotent.
  pay: (id, token) => wire(() => api.post<unknown>(`orders/${encodeURIComponent(id)}/payment`, undefined, { "X-Order-Token": token })),
};

/** The `error` word of a refusal's body, or "" when it has none. */
function codeOf(body: unknown): string {
  const code = typeof body === "object" && body !== null ? (body as { error?: unknown }).error : undefined;
  return typeof code === "string" ? code : "";
}

/** How often the shop is asked again while a payment is being confirmed, and for how long. */
export const POLL_MS = 3_000;
export const GIVE_UP_MS = 30_000;

/** Where the browser goes: one place, so a test can watch it leave. */
export const navigation = {
  to(url: string): void {
    window.location.assign(url);
  },
};

const REFERENCE = /^(DOI-\d{1,15})-\d{1,9}$/i;

/** The order number inside a reference (`DOI-24818-2` → `DOI-24818`), or null when it is not one. */
export function orderNumberOf(reference: string | null): string | null {
  const found = REFERENCE.exec((reference ?? "").trim());
  return found ? found[1].toUpperCase() : null;
}

/** The stored order of that number, with its token: the only token this page ever tries. */
export function heldFor(number: string | null): StoredOrder | null {
  if (number === null) return null;
  return storedOrders().find((o) => typeof o.token === "string" && String(o.number ?? "").toUpperCase() === number) ?? null;
}

function orderOf(body: unknown): OrderState | null {
  if (typeof body !== "object" || body === null) return null;
  const o = body as Record<string, unknown>;
  if (typeof o.id !== "string" || typeof o.status !== "string" || typeof o.paid !== "boolean") return null;
  return { id: o.id, number: typeof o.number === "string" ? o.number : "", status: o.status, paid: o.paid };
}

/** A call that never throws: no answer is status 0. */
async function call(run: () => Promise<Wire>): Promise<Wire> {
  try {
    return await run();
  } catch {
    return { status: 0, body: undefined };
  }
}

export type Read = { kind: "order"; order: OrderState } | { kind: "no-match" } | { kind: "failed" };

/**
 * Asks the shop what the order is. Only the shop's own 404 (code not_found) is
 * "this browser's token opens nothing"; any other failure, another 404 included (a
 * gateway's own page, a mis-routed address), is "could not check".
 */
export async function readOrder(held: StoredOrder): Promise<Read> {
  const { status, body } = await call(() => transport.read(held.id, held.token));
  if (status === 404 && codeOf(body) === "not_found") return { kind: "no-match" };
  const order = status === 200 ? orderOf(body) : null;
  return order ? { kind: "order", order } : { kind: "failed" };
}

export type Again = { kind: "url"; url: string } | { kind: "conflict" } | { kind: "failed" };

/** "Try again": a fresh VNPay address for the order, or why there is none. */
export async function payAgain(held: StoredOrder): Promise<Again> {
  const { status, body } = await call(() => transport.pay(held.id, held.token));
  if (status === 409) return { kind: "conflict" };
  const url = status === 200 && typeof body === "object" && body !== null ? (body as { payment_url?: unknown }).payment_url : undefined;
  const safe = typeof url === "string" ? payAddress(url) : null;
  return safe ? { kind: "url", url: safe } : { kind: "failed" };
}

/**
 * What to say of an order the shop answered for. Cancelled beats paid (a payment
 * that arrived after the order was given up is the dealer's to refund), paid beats
 * confirming, and an unpaid order is "confirming" only while the address says
 * success; any other code, or none, is "not completed".
 */
export function decide(order: OrderState, code: string | null, timedOut: boolean): PaymentState {
  if (order.status === "cancelled") return "cancelled";
  if (order.paid) return "paid";
  if (code !== "00") return "not-completed";
  return timedOut ? "timed-out" : "confirming";
}

/**
 * Asks the shop now and, while the page is confirming, every 3 seconds for up
 * to 30 seconds from opening. After the first answer a failed ask changes
 * nothing (what is known stays); before it, the failure is the answer.
 * Returns the way to stop.
 */
function watch(held: StoredOrder, code: string | null, show: (r: Read) => void, giveUp: () => void): () => void {
  let open = true;
  let polling = true;
  let timer: ReturnType<typeof setTimeout> | undefined;
  const stop = setTimeout(() => {
    polling = false;
    clearTimeout(timer);
    giveUp();
  }, GIVE_UP_MS);
  const step = async (known: OrderState | null): Promise<void> => {
    const r = await readOrder(held);
    if (!open) return;
    const order = r.kind === "order" ? r.order : known;
    if (r.kind === "order" || known === null) show(r);
    if (polling && order && decide(order, code, false) === "confirming") {
      timer = setTimeout(() => void step(order), POLL_MS);
    }
  };
  void step(null);
  return () => {
    open = false;
    clearTimeout(timer);
    clearTimeout(stop);
  };
}

export interface PaymentReturn {
  /** `loading` until the shop has answered the first time. */
  state: PaymentState | "loading";
  /** The order this browser holds a token for, when the reference is of one. */
  order: { id: string; number: string } | null;
  trying: boolean;
  /** "Try again" could not open the payment page. */
  tryFailed: boolean;
  again: () => void;
}

type Phase = { kind: "loading" } | Read;

function stateOf(phase: Phase, code: string | null, timedOut: boolean): PaymentState | "loading" {
  if (phase.kind === "loading") return "loading";
  if (phase.kind === "no-match") return "no-match";
  if (phase.kind === "failed") return "check-failed";
  return decide(phase.order, code, timedOut);
}

/** What the page shows for the address's reference and response code. */
export function usePaymentReturn(reference: string | null, code: string | null): PaymentReturn {
  const held = useMemo(() => heldFor(orderNumberOf(reference)), [reference]);
  const [phase, setPhase] = useState<Phase>(held ? { kind: "loading" } : { kind: "no-match" });
  const [timedOut, setTimedOut] = useState(false);
  const [trying, setTrying] = useState(false);
  const [tryFailed, setTryFailed] = useState(false);

  useEffect(() => {
    if (!held) return;
    return watch(held, code, setPhase, () => setTimedOut(true));
  }, [held, code]);

  // Back from VNPay's page with the browser's cache: the button is pressable again.
  useEffect(() => {
    const back = (e: PageTransitionEvent) => {
      if (e.persisted) setTrying(false);
    };
    window.addEventListener("pageshow", back);
    return () => window.removeEventListener("pageshow", back);
  }, []);

  const again = useCallback(() => {
    if (!held || trying) return;
    setTrying(true);
    setTryFailed(false);
    void (async () => {
      const a = await payAgain(held);
      if (a.kind === "url") return navigation.to(a.url);
      if (a.kind === "conflict") {
        // Paid or given up in the meantime: ask again and draw what it now is.
        const r = await readOrder(held);
        if (r.kind === "order") setPhase(r);
        setTryFailed(r.kind !== "order" || (!r.order.paid && r.order.status !== "cancelled"));
      } else setTryFailed(true);
      setTrying(false);
    })();
  }, [held, trying]);

  // When the shop says it knows nothing of the token, nothing about any order is drawn.
  const order = held && phase.kind !== "no-match" ? { id: held.id, number: held.number } : null;
  return { state: stateOf(phase, code, timedOut), order, trying, tryFailed, again };
}
