import { useSyncExternalStore } from "react";
import type { CartLine } from "../cart";
import type { OrderStatus } from "../shop";

/**
 * THE DESIGN'S MOCK ORDERS: what a visitor placed at checkout, kept on this
 * browser so the order page and the tracking page can find it. The orders
 * the shop already has are records in src/data/orders.json; these are the
 * new ones. After approval the backend story replaces this module: an order
 * is created by the API and marked paid only by the payment provider's own
 * notice (VNPay, MoMo), never by the browser.
 */
export interface PlacedOrder {
  id: string;
  number: string;
  placed_at: string;
  status: OrderStatus;
  lines: CartLine[];
  subtotal: number;
  delivery_fee: number;
  total: number;
  delivery: string;
  payment: string;
  paid: boolean;
  customer: string;
  phone: string;
  address: string;
  city: string;
  note: string;
  eta: string;
}

const KEY = "doi:orders";
let memory: string | null = null;
const listeners = new Set<() => void>();

function load(): string | null {
  try {
    return localStorage.getItem(KEY);
  } catch {
    return memory;
  }
}

let lastRaw: string | null | undefined;
let last: PlacedOrder[] = [];

function read(): PlacedOrder[] {
  const raw = load();
  if (raw !== lastRaw) {
    lastRaw = raw;
    try {
      const v = raw ? (JSON.parse(raw) as unknown) : [];
      last = Array.isArray(v) ? (v as PlacedOrder[]) : [];
    } catch {
      last = [];
    }
  }
  return last;
}

function write(orders: PlacedOrder[]) {
  const raw = JSON.stringify(orders);
  memory = raw;
  try {
    localStorage.setItem(KEY, raw);
  } catch {
    // Storage refused: the order lasts this page.
  }
  listeners.forEach((l) => l());
}

function addDays(from: Date, days: number): string {
  const d = new Date(from);
  d.setDate(d.getDate() + days);
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`;
}

export interface NewOrder {
  lines: CartLine[];
  delivery: string;
  delivery_fee: number;
  delivery_days: number;
  payment: string;
  online: boolean;
  customer: string;
  phone: string;
  address: string;
  city: string;
  note: string;
}

/** Places an order on this browser and returns it. An online method is paid at once in the mock. */
export function placeOrder(input: NewOrder): PlacedOrder {
  const now = new Date();
  const number = `DOI-${String(24818 + read().length + Math.floor(Math.random() * 40))}`;
  const subtotal = input.lines.reduce((s, l) => s + l.price * l.qty, 0);
  const order: PlacedOrder = {
    id: number.toLowerCase(),
    number,
    placed_at: now.toISOString(),
    status: "placed",
    lines: input.lines,
    subtotal,
    delivery_fee: input.delivery_fee,
    total: subtotal + input.delivery_fee,
    delivery: input.delivery,
    payment: input.payment,
    paid: input.online,
    customer: input.customer,
    phone: input.phone,
    address: input.address,
    city: input.city,
    note: input.note,
    eta: addDays(now, input.delivery_days),
  };
  write([order, ...read().filter((o) => o.id !== order.id)]);
  return order;
}

function subscribe(l: () => void) {
  listeners.add(l);
  return () => {
    listeners.delete(l);
  };
}

const NONE: PlacedOrder[] = [];

/** The orders placed on this browser, newest first. */
export function usePlacedOrders(): PlacedOrder[] {
  return useSyncExternalStore(subscribe, read, () => NONE);
}
