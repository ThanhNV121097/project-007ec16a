import type { PaymentOption } from "../orders";

/**
 * STANDS IN FOR THREE CALLS THE SHOP DOES NOT ANSWER YET: GET /v1/payment-options
 * listing bank transfer and MoMo, the transfer-details call of an order, and
 * PUT /v1/orders/{id} with {"paid": true}. Only src/lib/more-ways-to-pay-vietqr-
 * transfer-and-momo.ts imports it; the backend step deletes it and points that
 * one file at src/lib/api.ts. It writes nothing to the browser.
 *
 * Which answer comes back is asked for by the ADDRESS (`?mock=a,b`), never by a
 * control on the page, and with no `mock` in the address nothing here is used:
 *   options   all (three ways, all open)   no-vnpay (VNPay closed, bank, MoMo)
 *             closed (VNPay alone, closed)
 *   way       bank / momo / bank-paid: the order page's order pays that way
 *   transfer  slow (2 s)  error (500)  gone (503, the account was removed)
 *             long (the longest holder and account number)
 *   mark paid slow (2 s)  refused (422)  expired (401)
 *   any call  offline (nothing answers)
 *
 * The member names of the transfer details are this mock's: the technical
 * design fixes the real ones. The QR picture is a stand-in pattern, not a code
 * a banking app can read; the design decides who draws the real one.
 */
export interface Answer {
  status: number;
  body?: unknown;
}

/** The modes the address asks for. */
export function modes(): Set<string> {
  if (typeof window === "undefined") return new Set();
  const raw = new URLSearchParams(window.location.search).get("mock") ?? "";
  return new Set(raw.split(",").map((m) => m.trim()).filter(Boolean));
}

const wait = (ms: number) => new Promise<void>((done) => setTimeout(done, ms));

const VNPAY: PaymentOption = {
  id: "vnpay", name: "VNPay — ATM, Visa, Mastercard", icon: "card", online: true, is_available: true,
  note: "Pay by domestic ATM card, Visa, Mastercard or QR on VNPay's page. We pack your order after the payment is confirmed.",
};
const BANK: PaymentOption = {
  id: "bank", name: "Bank transfer (VietQR)", icon: "qr", online: true, is_available: true,
  note: "Scan the QR code on the next screen with any banking app. We send the pair as soon as the money lands.",
};
const MOMO: PaymentOption = {
  id: "momo", name: "MoMo wallet", icon: "wallet", online: true, is_available: true,
  note: "Confirm the payment in the MoMo app.",
};

/** The list the address asks for, or null when it asks for none and the shop is asked. */
export function mockOptions(): PaymentOption[] | null {
  const m = modes();
  if (m.has("all")) return [VNPAY, BANK, MOMO];
  if (m.has("no-vnpay")) return [{ ...VNPAY, is_available: false }, BANK, MOMO];
  if (m.has("closed")) return [{ ...VNPAY, is_available: false }];
  return null;
}

/** The order page's order, paying the way the address asks for. */
export function demoWay<T extends { payment: string; paid: boolean }>(order: T): T {
  const m = modes();
  if (m.has("bank-paid")) return { ...order, payment: "bank", paid: true };
  if (m.has("bank")) return { ...order, payment: "bank", paid: false };
  if (m.has("momo")) return { ...order, payment: "momo", paid: false };
  return order;
}

export async function getTransfer(number: string, total: number): Promise<Answer> {
  const m = modes();
  if (m.has("offline")) throw new TypeError("Failed to fetch");
  await wait(m.has("slow") ? 2000 : 150);
  if (m.has("error")) return { status: 500, body: { error: "internal", message: "Something went wrong." } };
  if (m.has("gone")) return { status: 503, body: { error: "payment_unavailable", message: "Bank transfer is not available." } };
  const long = m.has("long");
  return {
    status: 200,
    body: {
      holder: long ? "CONG TY TRACH NHIEM HUU HAN THUONG MAI VA DICH VU GIAY DOI VIET NAM" : "NGUYEN VAN AN",
      bank: long ? "Ngan hang Thuong mai Co phan Ngoai thuong Viet Nam (Vietcombank)" : "Vietcombank",
      account_number: long ? "0071001234567890123" : "0071001234567",
      amount: total,
      note: number,
      payload: `000201010212385700A0000007270127000697043601130071001234567020801QRIBFTTA5303704540${total}5802VN62${number}6304`,
    },
  };
}

export async function putPaid(id: string, status: string): Promise<Answer> {
  const m = modes();
  if (m.has("offline")) throw new TypeError("Failed to fetch");
  await wait(m.has("slow") ? 2000 : 400);
  if (m.has("expired")) return { status: 401, body: { error: "unauthorized", message: "Sign in." } };
  if (m.has("refused")) {
    return { status: 422, body: { error: "invalid", message: "Refused.", errors: { paid: "Only the payment provider marks this order paid." } } };
  }
  return { status: 200, body: { id, paid: true, status: status === "placed" ? "confirmed" : status } };
}

const SIZE = 33;

/** One module of a finder square whose corner is at (0, 0); null outside it and its light margin. */
function finderAt(r: number, c: number): boolean | null {
  if (r < -1 || r > 7 || c < -1 || c > 7) return null;
  if (r === -1 || r === 7 || c === -1 || c === 7) return false;
  return r === 0 || r === 6 || c === 0 || c === 6 || (r >= 2 && r <= 4 && c >= 2 && c <= 4);
}

/** A stand-in picture of the payload: three finder squares and a pattern seeded from the text. */
export function standInModules(payload: string): boolean[][] {
  let seed = 2166136261;
  for (let i = 0; i < payload.length; i++) seed = Math.imul(seed ^ payload.charCodeAt(i), 16777619) >>> 0;
  const next = () => {
    seed ^= seed << 13;
    seed >>>= 0;
    seed ^= seed >>> 17;
    seed ^= seed << 5;
    seed >>>= 0;
    return seed;
  };
  const far = SIZE - 7;
  const rows: boolean[][] = [];
  for (let r = 0; r < SIZE; r++) {
    const row: boolean[] = [];
    for (let c = 0; c < SIZE; c++) {
      const hit = [finderAt(r, c), finderAt(r, c - far), finderAt(r - far, c)].find((v) => v !== null);
      row.push(hit !== undefined ? hit : next() % 2 === 0);
    }
    rows.push(row);
  }
  return rows;
}
