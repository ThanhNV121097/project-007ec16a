import { vi } from "vitest";
import { api } from "../lib/api";
import type { PublicOrder } from "../lib/order-page-and-pay-now";

// What the Order page's tests stub: the shop's answers through the api client
// (each test file mocks ../lib/api before it imports this one).

export const pair = { product: "samba-og-w-core-black", name: "Samba OG Black Crystal", colourway: "Core Black", image: "/images/ref/samba-og-w-core-black.jpg", size: "38", qty: 1, unit_price: 3290000 };
export const kids = { product: "samba-og-kids-cloud-white", name: "Samba OG Kids Cloud White", colourway: "Cloud White", image: "/images/ref/samba-og-kids-cloud-white.jpg", size: "36", qty: 2, unit_price: 2290000 };

export const sample: PublicOrder = {
  id: "doi-24818", number: "DOI-24818", placed_at: "2026-11-25T10:00:00+07:00", status: "placed", items: [pair],
  total: 3290000, payment: "vnpay", paid: false, delivery: "standard", carrier: "", tracking: "", eta: "2026-11-28",
  customer: "Lan Nguyen", city: "Da Nang",
};

const list = (items: unknown[]) => ({ items, total: items.length, page: 1, limit: 10 });

/** The shop's reads: the two name lists as the records serve them, and `order` for GET /v1/orders/{id}. */
export function serve(order: (path: string, headers?: Record<string, string>) => unknown): void {
  vi.mocked(api.get).mockImplementation((async (path: string, _params?: unknown, headers?: Record<string, string>) => {
    if (path === "payments") return list([{ id: "vnpay", name: "VNPay — ATM, Visa, Mastercard" }]);
    if (path === "delivery") return list([{ id: "standard", name: "Standard", days: "2–4 working days", fee: 0 }]);
    return order(path, headers);
  }) as typeof api.get);
}

/** The order's read, answered with this order (any fields added to it travel as the shop sent them). */
export function serveOrder(over: Record<string, unknown> = {}): void {
  serve(() => ({ ...sample, ...over }));
}

/** Gives this browser the token of an order, the way placing it does. */
export function holdToken(id: string, token: string): void {
  localStorage.setItem("doi:placed", JSON.stringify([{ id, number: id.toUpperCase(), placed_at: sample.placed_at, token }]));
}
