import { ApiError } from "../api";
import type { DealerOrder, OrderPair, OrderPatch, Transport } from "../dealer-handles-orders-status-courier-payment";
import type { OrderStatus } from "../shop";

/**
 * STANDS IN FOR `GET /v1/orders/{id}`, `PUT /v1/orders/{id}` AND
 * `DELETE /v1/orders/{id}` WHILE THE BACKEND IS NOT BUILT. Deleted with it.
 *
 * It judges as the shop will: one step along placed → delivered, cancel from
 * placed, confirmed or packed, nothing but the note on a delivered or
 * cancelled order, `paid` never, a cancelled and unpaid order alone may go.
 * It lives in this page's memory only and writes nothing to the browser.
 *
 * Answers on demand, by the address (`?mock=`), never by a control on the page:
 * `slow` (2 s), `offline` (writes are not answered), `expired` (writes are 401).
 */

const SHOES: Record<string, Pick<OrderPair, "name" | "colourway" | "image">> = {
  "samba-og-w-core-black": { name: "Samba OG Core Black", colourway: "Core Black", image: "/images/ref/samba-og-w-core-black.jpg" },
  "samba-og-w-crystal-white": { name: "Samba OG Crystal White", colourway: "Crystal White", image: "/images/ref/samba-og-w-crystal-white.jpg" },
  "handball-spezial-w-earth-strata": { name: "Handball Spezial Earth Strata", colourway: "Earth Strata", image: "/images/ref/handball-spezial-w-earth-strata.jpg" },
  "gazelle-indoor-solar-gold": { name: "Gazelle Indoor Solar Gold", colourway: "Solar Gold", image: "/images/ref/gazelle-indoor-solar-gold.jpg" },
  "adizero-evo-sl-charcoal": { name: "Adizero Evo SL Charcoal", colourway: "Charcoal", image: "/images/ref/adizero-evo-sl-charcoal.jpg" },
  "megaride-s2-cloud-white": { name: "Megaride S2 Cloud White", colourway: "Cloud White", image: "/images/ref/megaride-s2-cloud-white.jpg" },
};

function pair(product: string, size: string, qty: number, unit_price: number): OrderPair {
  return { product, ...SHOES[product], size, qty, unit_price };
}

function make(n: number, f: Partial<DealerOrder> & { pairs: OrderPair[] }): DealerOrder {
  const { pairs, ...rest } = f;
  const first = pairs[0];
  return {
    id: `doi-${n}`, number: `DOI-${n}`, placed_at: "", status: "placed", product: first.product, size: first.size,
    qty: first.qty, total: pairs.reduce((sum, p) => sum + p.unit_price * p.qty, 0), delivery_fee: 0, payment: "vnpay",
    paid: false, delivery: "standard", carrier: "", tracking: "", eta: "", customer: "", city: "", phone: "",
    email: "", address: "", note: "", transaction_no: "", items: pairs, ...rest,
  };
}

function seed(): DealerOrder[] {
  const core = (size: string, qty = 1) => pair("samba-og-w-core-black", size, qty, 2790000);
  return [
    make(24817, { placed_at: "2026-11-23T16:05:00+07:00", status: "shipped", payment: "momo", paid: true, carrier: "GHN", tracking: "GHN-PT27XM", eta: "2026-11-26", customer: "Hoàng Nam", phone: "0938 410 552", address: "42 Example Street, Hải Châu", city: "Đà Nẵng", note: "Gift — no price inside the box.", pairs: [pair("samba-og-w-crystal-white", "39 1/3", 1, 3090000)] }),
    make(24816, { placed_at: "2026-11-25T08:40:00+07:00", payment: "cod", eta: "2026-11-28", customer: "Trang", phone: "0977 302 118", address: "7 Example Lane", city: "Biên Hòa", pairs: [pair("handball-spezial-w-earth-strata", "37 1/3", 1, 3290000)] }),
    make(24815, { placed_at: "2026-11-24T10:12:00+07:00", status: "confirmed", paid: true, transaction_no: "14123457", carrier: "GHN", eta: "2026-11-25", customer: "Long", phone: "0912 774 630", email: "long@example.com", address: "15 Example Road, Ba Đình", city: "Hà Nội", pairs: [pair("handball-spezial-w-earth-strata", "37 1/3", 1, 3290000), pair("samba-og-w-core-black", "38", 1, 2600000)] }),
    make(24814, { placed_at: "2026-11-23T19:30:00+07:00", status: "packed", payment: "cod", carrier: "GHTK", eta: "2026-11-27", customer: "Hương", phone: "0905 661 207", address: "3 Example Street", city: "Vũng Tàu", pairs: [pair("gazelle-indoor-solar-gold", "40", 1, 2290000)] }),
    make(24813, { placed_at: "2026-11-22T11:02:00+07:00", status: "shipped", payment: "momo", paid: true, carrier: "GHN", tracking: "GHN-KQ81DA", eta: "2026-11-25", customer: "Tuấn", phone: "0983 215 904", address: "21 Example Street, Cầu Giấy", city: "Hà Nội", pairs: [pair("adizero-evo-sl-charcoal", "42", 1, 3590000)] }),
    make(24812, { placed_at: "2026-11-21T14:48:00+07:00", status: "shipped", payment: "bank", paid: true, carrier: "GHTK", tracking: "GHTK-5520193", eta: "2026-11-26", customer: "Mai Linh", phone: "0934 880 312", address: "9 Example Avenue", city: "Nha Trang", pairs: [core("38", 2)] }),
    make(24811, { placed_at: "2026-11-20T09:20:00+07:00", status: "cancelled", payment: "cod", customer: "Đức", phone: "0919 040 776", address: "5 Example Street", city: "Huế", note: "Customer cancelled by phone; ordered 43 1/3 instead in store.", pairs: [pair("megaride-s2-cloud-white", "43 1/3", 1, 4090000)] }),
    make(24810, { placed_at: "2026-11-19T21:15:00+07:00", status: "delivered", paid: true, transaction_no: "14123456", carrier: "GHN", tracking: "GHN-MW55RB", eta: "2026-11-20", customer: "Phương", phone: "0908 552 671", address: "88 Example Street, Quận 3", city: "TP. Hồ Chí Minh", pairs: [pair("handball-spezial-w-earth-strata", "37 1/3", 1, 3290000)] }),
    make(24809, { placed_at: "2026-11-18T13:44:00+07:00", status: "delivered", payment: "momo", paid: true, carrier: "GHN", tracking: "GHN-HC09TT", eta: "2026-11-21", customer: "Khánh", phone: "0937 118 249", address: "12 Example Road, Bình Thạnh", city: "TP. Hồ Chí Minh", pairs: [pair("adizero-evo-sl-charcoal", "41 1/3", 1, 5090000)] }),
    make(24808, { placed_at: "2026-11-17T08:05:00+07:00", status: "delivered", payment: "cod", paid: true, carrier: "GHTK", tracking: "GHTK-5518840", eta: "2026-11-20", customer: "Ngọc", phone: "0915 330 482", address: "30 Example Street", city: "Hải Phòng", pairs: [pair("gazelle-indoor-solar-gold", "39 1/3", 1, 2290000)] }),
    make(24807, { placed_at: "2026-11-15T17:36:00+07:00", status: "delivered", payment: "bank", paid: true, carrier: "GHN", tracking: "GHN-AZ71PE", eta: "2026-11-19", customer: "Bảo", phone: "0939 602 115", address: "6 Example Street, Ninh Kiều", city: "Cần Thơ", pairs: [pair("megaride-s2-cloud-white", "43", 1, 3790000)] }),
    make(24806, { placed_at: "2026-11-14T10:58:00+07:00", status: "delivered", paid: true, transaction_no: "14123455", carrier: "GHN", tracking: "GHN-RR42LN", eta: "2026-11-15", customer: "Thu Hà", phone: "0904 771 390", address: "18 Example Lane, Hoàn Kiếm", city: "Hà Nội", pairs: [pair("samba-og-w-crystal-white", "38", 1, 3090000)] }),
    make(24805, { placed_at: "2026-11-13T15:21:00+07:00", status: "delivered", payment: "momo", paid: true, carrier: "GHTK", tracking: "GHTK-5507712", eta: "2026-11-16", customer: "Quang", phone: "0935 248 016", address: "27 Example Street, Sơn Trà", city: "Đà Nẵng", pairs: [pair("adizero-evo-sl-charcoal", "42 2/3", 1, 3590000)] }),
    make(24804, { placed_at: "2026-11-12T09:14:00+07:00", status: "delivered", payment: "cod", paid: true, carrier: "GHN", tracking: "GHN-LJ8K4Q", eta: "2026-11-15", customer: "Minh Anh", phone: "0903 118 274", address: "64 Example Street, Đống Đa", city: "Hà Nội", pairs: [core("38")] }),
  ];
}

let orders = seed();

/** Puts the sample orders back as they were. For the tests. */
export function resetMock(): void {
  orders = seed();
}

const MOVES: Record<OrderStatus, readonly OrderStatus[]> = {
  placed: ["confirmed", "cancelled"],
  confirmed: ["packed", "cancelled"],
  packed: ["shipped", "cancelled"],
  shipped: ["delivered"],
  delivered: [],
  cancelled: [],
};
const NEVER = ["number", "placed_at", "product", "size", "qty", "items", "total", "delivery_fee", "payment", "delivery", "transaction_no"];
const DEALER = ["status", "carrier", "tracking", "eta", "customer", "phone", "email", "address", "city", "note"];
const WHILE_OPEN = ["carrier", "tracking", "eta", "customer", "phone", "email", "address", "city"];

const pause = (ms: number) => new Promise<void>((resolve) => setTimeout(resolve, ms));

function wanted(): string {
  return typeof window === "undefined" ? "" : (new URLSearchParams(window.location.search).get("mock") ?? "");
}

async function arrive(write: boolean): Promise<void> {
  const how = wanted();
  await pause(how === "slow" ? 2000 : 0);
  if (write && how === "offline") throw new ApiError(0, "network", "The server cannot be reached.");
  if (write && how === "expired") throw new ApiError(401, "unauthenticated", "Sign in to continue.");
}

function find(id: string): number {
  const at = orders.findIndex((o) => o.id === id);
  if (at < 0) throw new ApiError(404, "not_found", "That order is not here.");
  return at;
}

function validDay(v: string): boolean {
  const m = /^(\d{4})-(\d{2})-(\d{2})$/.exec(v);
  if (!m) return false;
  const d = new Date(Date.UTC(Number(m[1]), Number(m[2]) - 1, Number(m[3])));
  return d.getUTCMonth() === Number(m[2]) - 1 && d.getUTCDate() === Number(m[3]);
}

function statusError(from: OrderStatus, to: unknown): string | undefined {
  if (typeof to !== "string" || !(to in MOVES)) return "Status must be placed, confirmed, packed, shipped, delivered or cancelled.";
  if (to === from && MOVES[from].length > 0) return undefined;
  return MOVES[from].includes(to as OrderStatus) ? undefined : `An order cannot go from ${from} to ${to}.`;
}

function fieldError(key: string, value: unknown): string | undefined {
  if (typeof value !== "string") return "This must be text.";
  if (key === "eta") return value === "" || validDay(value) ? undefined : "Write the date as year-month-day, for example 2026-11-26.";
  if (key === "customer" && value.trim() === "") return "Enter the customer's name.";
  if (key === "phone" && value.replace(/\D/g, "").length < 9) return "Enter a phone number with at least 9 digits.";
  if (key === "email" && value !== "" && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value)) return "Enter a valid email address, or leave it empty.";
  if (key === "note") return value.length > 5000 ? "Keep the note to 5,000 characters." : undefined;
  return value.length > 200 ? "Keep this to 200 characters." : undefined;
}

function refusals(order: DealerOrder, body: Record<string, unknown>): Record<string, string> {
  const closed = MOVES[order.status].length === 0;
  const errors: Record<string, string> = {};
  for (const [key, value] of Object.entries(body)) {
    if (key === "id") continue;
    if (key === "paid") errors[key] = "Only the payment provider marks this order paid.";
    else if (NEVER.includes(key)) errors[key] = "This cannot be changed.";
    else if (!DEALER.includes(key)) throw new ApiError(400, "bad_request", "That field does not exist.");
    else if (key === "status") {
      const said = statusError(order.status, value);
      if (said) errors[key] = said;
    } else if (closed && WHILE_OPEN.includes(key)) errors[key] = `An order that is ${order.status} cannot be changed.`;
    else {
      const said = fieldError(key, value);
      if (said) errors[key] = said;
    }
  }
  return errors;
}

async function read(id: string): Promise<DealerOrder> {
  await arrive(false);
  return structuredClone(orders[find(id)]);
}

async function update(id: string, body: OrderPatch): Promise<DealerOrder> {
  await arrive(true);
  const at = find(id);
  const errors = refusals(orders[at], body as Record<string, unknown>);
  if (Object.keys(errors).length > 0) throw new ApiError(422, "invalid", "These details cannot be saved.", errors);
  orders[at] = { ...orders[at], ...body };
  return structuredClone(orders[at]);
}

async function remove(id: string): Promise<void> {
  await arrive(true);
  const order = orders[find(id)];
  if (order.status !== "cancelled") throw new ApiError(409, "conflict", "Cancel the order first.");
  if (order.paid) throw new ApiError(409, "conflict", "A paid order cannot be deleted; refund it in VNPay's portal first.");
  orders = orders.filter((o) => o.id !== id);
}

export const transport: Transport = { read, update, remove };
