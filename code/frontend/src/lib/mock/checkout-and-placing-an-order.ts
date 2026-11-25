import { collections } from "../../data";
import type { CartLine } from "../cart";
import type { OrderRequest, PaymentOption, Wire } from "../orders";
import type { Product } from "../shop";
import { placeOrder as keepForOrderPage } from "./orders";

/**
 * THE CHECKOUT'S MOCK SHOP. Stands in for GET /v1/payment-options and
 * POST /v1/orders (docs/architecture/services.md §3.5, §3.10) until the
 * backend story: src/lib/orders.ts is the only file that imports it, and the
 * backend step deletes it and points that file at src/lib/api.ts.
 *
 * It answers as the API will: it reads every price from the shoe records
 * (never from the request), refuses a pair that is sold out, short or no
 * longer live with every failing pair named, and answers a repeated
 * Idempotency-Key with the first order and a new token.
 *
 * The states the design never drew are driven from here, not from the page:
 * localStorage "doi:mock" = "server-error" answers 500, "payment-off" reports
 * VNPay unavailable. A size in a shoe's last-pairs list counts as two pairs.
 *
 * The Order page is still the design's mock (it reads src/lib/mock/orders.ts
 * until "Order page and Pay now"), so each order this shop accepts is also
 * kept there, unpaid, for the page the browser lands on to show. That bridge
 * goes with this module; the browser's own store of placed orders is
 * src/lib/orders.ts and never reads it.
 */
const LOW_LEFT = 2;
const DELAY = 900;

function scenario(): "ok" | "server-error" | "payment-off" {
  try {
    const v = localStorage.getItem("doi:mock");
    return v === "server-error" || v === "payment-off" ? v : "ok";
  } catch {
    return "ok";
  }
}

export async function getPaymentOptions(): Promise<{ items: PaymentOption[]; total: number; page: number; limit: number }> {
  await Promise.resolve();
  const vnpay: PaymentOption = {
    id: "vnpay",
    name: "VNPay — ATM, Visa, Mastercard",
    note: "Pay by card or banking app on VNPay's secure page.",
    icon: "card",
    online: true,
    is_available: scenario() !== "payment-off",
  };
  return { items: [vnpay], total: 1, page: 1, limit: 100 };
}

const seen = new Map<string, { phone: string; order: unknown }>();

const digits = (s: string) => s.replace(/\D/g, "");

function newToken(): string {
  return Array.from(crypto.getRandomValues(new Uint8Array(24)), (b) => b.toString(16).padStart(2, "0")).join("");
}

function refusals(req: OrderRequest, shoes: Product[]): Record<string, string> {
  const out: Record<string, string> = {};
  req.items.forEach((it, i) => {
    const p = shoes.find((s) => s.id === it.product);
    const label = `${p?.name ?? it.product} · EU ${it.size}`;
    if (!p || p.status !== "live") out[`items.${i}`] = `${label} — no longer in the shop.`;
    else if ((p.sizes_out ?? []).includes(it.size)) out[`items.${i}`] = `${label} — sold out in your size.`;
    else if ((p.sizes_low ?? []).includes(it.size) && it.qty > LOW_LEFT) out[`items.${i}`] = `${label} — only ${LOW_LEFT} left in your size.`;
  });
  return out;
}

export async function postOrder(req: OrderRequest, key: string): Promise<Wire> {
  await new Promise<void>((r) => window.setTimeout(r, DELAY));
  const mode = scenario();
  if (mode === "server-error") return { status: 500, body: { error: "internal", message: "Something went wrong." } };
  if (mode === "payment-off") return { status: 503, body: { error: "payment_unavailable", message: "Online payment is not available." } };
  const first = seen.get(key);
  if (first) {
    if (digits(first.phone) !== digits(req.phone)) return { status: 409, body: { error: "conflict", message: "That key was used for another order." } };
    return { status: 200, body: { order: first.order, token: newToken(), payment_url: "" } };
  }
  const shoes = (collections().products ?? []) as unknown as Product[];
  const errors = refusals(req, shoes);
  if (Object.keys(errors).length > 0) return { status: 409, body: { error: "stock_conflict", message: "Some pairs are not available.", errors } };
  const lines: CartLine[] = req.items.map((it) => {
    const p = shoes.find((s) => s.id === it.product);
    return { id: it.product, name: p?.name ?? it.product, price: p?.price ?? 0, image: p?.image, variant: `EU ${it.size}`, qty: it.qty };
  });
  const kept = keepForOrderPage({ lines, payment: "vnpay", online: false, customer: req.customer, phone: req.phone, address: req.address, city: req.city, note: req.note });
  const order = {
    id: kept.id, number: kept.number, placed_at: kept.placed_at, status: kept.status, paid: kept.paid, total: kept.total,
    delivery_fee: 0, delivery: kept.delivery, payment: kept.payment,
  };
  seen.set(key, { phone: req.phone, order });
  return { status: 201, body: { order, token: newToken(), payment_url: "" } };
}
