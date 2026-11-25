import type { OrderItem, PublicOrder } from "../order-page-and-pay-now";

/**
 * THE ORDER PAGE'S MOCK: what GET /v1/orders/{id} and POST /v1/orders/{id}/payment
 * answer, as { status, body } the way a JSON API does (services.md §2.8, §3.11,
 * §3.13). Only src/lib/order-page-and-pay-now.ts imports it; the backend step
 * deletes it and points that one file at src/lib/api.ts.
 *
 * Which answer an address gets, so every state of the page can be reached by
 * address and not by a control on the page:
 *   /orders/doi-24818 …      a sample order (see SAMPLES)
 *   /orders/<placed here>    an order the mock checkout placed on this browser
 *   /orders/doi-slow         the same order after 2 seconds
 *   /orders/doi-error        500          /orders/doi-offline   no network
 *   /orders/doi-flaky        500 once, then the order
 *   anything else            404, the one body of a missing order
 * "Pay now" answers 409 "Online payment is not available yet." for every order,
 * as the shop does until the VNPay story lands.
 */
export interface Answer {
  status: number;
  body?: unknown;
}

const pair = (product: string, name: string, colourway: string, size: string, qty: number, unit_price: number): OrderItem => ({
  product, name, colourway, image: `/images/ref/${product}.jpg`, size, qty, unit_price,
});

const black = pair("samba-og-w-core-black", "Samba OG Black Crystal", "Core Black", "38", 1, 3290000);
const kids = pair("samba-og-kids-cloud-white", "Samba OG Kids Cloud White", "Cloud White", "36", 2, 2290000);
const crystal = pair("samba-og-w-crystal-white", "Samba OG Crystal White", "Crystal White", "39 1/3", 1, 3090000);
const gazelle = pair("gazelle-indoor-solar-gold", "Gazelle Indoor Solar Gold", "Solar Gold", "42", 1, 3590000);

function order(id: string, over: Partial<PublicOrder>): PublicOrder {
  const items = over.items ?? [black];
  return {
    id, number: id.toUpperCase(), placed_at: "2026-11-25T10:00:00+07:00", status: "placed", items,
    total: items.reduce((sum, i) => sum + i.unit_price * i.qty, 0),
    payment: "vnpay", paid: false, delivery: "standard", carrier: "", tracking: "", eta: "",
    customer: "Lan Nguyen", city: "Da Nang", ...over,
  };
}

const SAMPLES: PublicOrder[] = [
  order("doi-24818", { eta: "2026-11-28" }),
  order("doi-24819", { status: "packed", items: [black, kids], carrier: "GHTK", eta: "2026-11-27" }),
  order("doi-24817", { status: "shipped", items: [crystal], paid: true, carrier: "GHN", tracking: "GHN-PT27XM", eta: "2026-11-26", customer: "Hoàng Nam", city: "Đà Nẵng", placed_at: "2026-11-23T16:05:00+07:00" }),
  order("doi-24811", { status: "cancelled", items: [gazelle], customer: "Đức", city: "Huế", placed_at: "2026-11-20T09:20:00+07:00" }),
  order("doi-24810", { status: "delivered", paid: true, carrier: "GHN", tracking: "GHN-MW55RB", eta: "2026-11-20", customer: "Phương", city: "TP. Hồ Chí Minh", placed_at: "2026-11-19T21:15:00+07:00" }),
];

/** What the mock checkout keeps on this browser (src/lib/mock/orders.ts, key doi:orders), read as it is stored. */
interface Placed {
  id: string; number: string; placed_at: string; status: PublicOrder["status"]; total: number;
  lines: { id: string; name: string; price: number; image?: string; variant?: string; qty: number }[];
  delivery: string; payment: string; paid: boolean; customer: string; city: string; eta: string;
}

function placedHere(id: string): PublicOrder | undefined {
  try {
    const all = JSON.parse(localStorage.getItem("doi:orders") ?? "[]") as Placed[];
    const p = Array.isArray(all) ? all.find((o) => o.id === id) : undefined;
    if (!p) return undefined;
    return order(p.id, {
      number: p.number, placed_at: p.placed_at, status: p.status, total: p.total, delivery: p.delivery, payment: p.payment,
      paid: p.paid, customer: p.customer, city: p.city, eta: p.eta,
      items: p.lines.map((l) => ({ product: l.id, name: l.name, colourway: "", image: l.image ?? "", size: (l.variant ?? "").replace(/^EU\s*/, ""), qty: l.qty, unit_price: l.price })),
    });
  } catch {
    return undefined;
  }
}

const wait = (ms: number) => new Promise<void>((done) => setTimeout(done, ms));
const ALIAS: Record<string, string> = { "doi-slow": "doi-24818", "doi-flaky": "doi-24818" };
const GONE: Answer = { status: 404, body: { error: "not_found", message: "We can't find that." } };
let flaky = 0;

function find(id: string): PublicOrder | undefined {
  const found = placedHere(id) ?? SAMPLES.find((s) => s.id === (ALIAS[id] ?? id));
  return found && { ...found, id };
}

export async function getOrder(id: string): Promise<Answer> {
  if (id === "doi-offline") throw new TypeError("Failed to fetch");
  await wait(id === "doi-slow" ? 2000 : 120);
  if (id === "doi-error" || (id === "doi-flaky" && flaky++ === 0)) return { status: 500, body: { error: "internal", message: "Something went wrong." } };
  const found = find(id);
  return found ? { status: 200, body: found } : GONE;
}

export async function postPayment(id: string): Promise<Answer> {
  await wait(600);
  return find(id) ? { status: 409, body: { error: "conflict", message: "Online payment is not available yet." } } : GONE;
}
