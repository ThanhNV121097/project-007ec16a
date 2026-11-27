import { useCallback, useEffect, useState } from "react";
import { useContent } from "../editable";
import { api, ApiError } from "./api";
import type { OrderStatus } from "./shop";

/**
 * The dealer's one door to an order (services §2.8, §3.11, §3.15, §3.16).
 *
 * Everything the order page reads or writes goes through `transport`, three
 * calls on `/v1/orders/{id}` made with the project's one client, `lib/api.ts`.
 * Wire names are the API's.
 */

/** One pair of an order, as the shop sold it. */
export interface OrderPair {
  product: string;
  name: string;
  colourway: string;
  image: string;
  size: string;
  qty: number;
  unit_price: number;
}

/** The dealer view of an order: the public fields, the private ones and every pair. */
export interface DealerOrder {
  id: string;
  number: string;
  placed_at: string;
  status: OrderStatus;
  product: string;
  size: string;
  qty: number;
  total: number;
  delivery_fee: number;
  payment: string;
  paid: boolean;
  delivery: string;
  carrier: string;
  tracking: string;
  eta: string;
  customer: string;
  city: string;
  phone: string;
  email: string;
  address: string;
  note: string;
  transaction_no: string;
  items: OrderPair[];
}

/** The ten fields the dealer can edit, and the only ones a save may carry. */
export const EDITABLE = [
  "status", "carrier", "tracking", "eta", "customer", "phone", "email", "address", "city", "note",
] as const;

export type EditableKey = (typeof EDITABLE)[number];
export type OrderPatch = Partial<Pick<DealerOrder, EditableKey>>;
export type OrderValues = Record<EditableKey, string>;

export type Tab = "Order" | "Delivery" | "Customer";

/** What each editable field is called, and the tab it sits on. */
export const FIELD: Record<EditableKey, { label: string; tab: Tab }> = {
  status: { label: "Status", tab: "Order" },
  carrier: { label: "Carrier", tab: "Delivery" },
  tracking: { label: "Tracking number", tab: "Delivery" },
  eta: { label: "Expected", tab: "Delivery" },
  customer: { label: "Customer", tab: "Customer" },
  phone: { label: "Phone", tab: "Customer" },
  email: { label: "Email", tab: "Customer" },
  address: { label: "Address", tab: "Customer" },
  city: { label: "City or province", tab: "Customer" },
  note: { label: "Internal note", tab: "Customer" },
};

const MOVES: Record<OrderStatus, readonly OrderStatus[]> = {
  placed: ["confirmed", "cancelled"],
  confirmed: ["packed", "cancelled"],
  packed: ["shipped", "cancelled"],
  shipped: ["delivered"],
  delivered: [],
  cancelled: [],
};

/** The moves the shop allows from a status. The server is the judge; this only helps. */
export function movesFrom(status: OrderStatus): readonly OrderStatus[] {
  return MOVES[status];
}

/** Delivered and cancelled orders accept no change but their note. */
export function isLocked(status: OrderStatus): boolean {
  return MOVES[status].length === 0;
}

/** The status words the lists and the control use: Placed … On its way … Cancelled. */
export function useStatusLabels(): Record<OrderStatus, string> {
  const steps = useContent<Record<string, string>>("orders.steps");
  const cancelled = useContent("pages.admin.orders.cancelled");
  return {
    placed: steps.placed,
    confirmed: steps.confirmed,
    packed: steps.packed,
    shipped: steps.shipped,
    delivered: steps.delivered,
    cancelled,
  };
}

/** What an order's editable fields hold now, as the form's text boxes hold them. */
export function valuesOf(order: DealerOrder): OrderValues {
  const out = {} as OrderValues;
  for (const key of EDITABLE) out[key] = order[key] ?? "";
  return out;
}

/**
 * What a save carries: only the editable fields that differ from the order as
 * loaded, and the note alone for a delivered or cancelled order. Never the
 * whole record — the shop refuses `total`, `number` and the rest when present.
 */
export function changes(order: DealerOrder, values: OrderValues): OrderPatch {
  const allowed: readonly EditableKey[] = isLocked(order.status) ? ["note"] : EDITABLE;
  const out: Record<string, string> = {};
  for (const key of allowed) if (values[key] !== (order[key] ?? "")) out[key] = values[key];
  return out as OrderPatch;
}

/** The shop's answer to a read, or a reason there is none. */
export type OrderLoad =
  | { state: "loading" }
  | { state: "failed" }
  | { state: "missing" }
  | { state: "ready"; order: DealerOrder };

const at = (id: string) => `orders/${encodeURIComponent(id)}`;

/** The dealer's read of an order, a save of it (the dealer view answers) and its removal. */
const transport = {
  read: (id: string) => api.get<DealerOrder>(at(id)),
  update: (id: string, body: OrderPatch) => api.put<DealerOrder>(at(id), body),
  remove: (id: string) => api.delete(at(id)),
};

/** A record that is not there: the API's own `not_found`, never just any 404. */
function isNotFound(e: unknown): boolean {
  return e instanceof ApiError && e.status === 404 && e.code === "not_found";
}

/**
 * One order by id. A 404 is an answer ("missing"); no answer, a timeout and a
 * 5xx are "failed", so the missing view is never drawn for a shop that did not reply.
 */
export function useDealerOrder(id: string | undefined) {
  const [load, setLoad] = useState<OrderLoad>({ state: "loading" });
  const [attempt, setAttempt] = useState(0);

  useEffect(() => {
    if (id === undefined) {
      setLoad({ state: "missing" });
      return;
    }
    let live = true;
    setLoad({ state: "loading" });
    transport.read(id).then(
      (order) => {
        if (live) setLoad({ state: "ready", order });
      },
      (e: unknown) => {
        if (live) setLoad({ state: isNotFound(e) ? "missing" : "failed" });
      },
    );
    return () => {
      live = false;
    };
  }, [id, attempt]);

  const replace = useCallback((order: DealerOrder) => setLoad({ state: "ready", order }), []);
  const retry = useCallback(() => setAttempt((n) => n + 1), []);
  return { load, replace, retry };
}

/** What a save came to: done, refused by field, or never answered. */
export type SaveResult =
  | { ok: true; order: DealerOrder }
  | { ok: false; kind: "fields"; errors: Record<string, string> }
  | { ok: false; kind: "expired" }
  | { ok: false; kind: "unreachable" };

export type DeleteResult =
  | { ok: true }
  | { ok: false; kind: "refused"; message: string }
  | { ok: false; kind: "expired" }
  | { ok: false; kind: "unreachable" };

function expired(e: unknown): boolean {
  return e instanceof ApiError && e.status === 401;
}

/** A 422 names fields; a 401 is a session that ended; anything else is "could not save". */
export async function saveOrder(id: string, patch: OrderPatch): Promise<SaveResult> {
  try {
    return { ok: true, order: await transport.update(id, patch) };
  } catch (e) {
    if (e instanceof ApiError && e.status === 422 && e.errors) return { ok: false, kind: "fields", errors: e.errors };
    return { ok: false, kind: expired(e) ? "expired" : "unreachable" };
  }
}

/** A 409 (or a 404 for an order already gone) carries a sentence for the open question. */
export async function deleteOrder(id: string): Promise<DeleteResult> {
  try {
    await transport.remove(id);
    return { ok: true };
  } catch (e) {
    if (e instanceof ApiError && (e.status === 409 || e.status === 404)) return { ok: false, kind: "refused", message: e.message };
    return { ok: false, kind: expired(e) ? "expired" : "unreachable" };
  }
}
