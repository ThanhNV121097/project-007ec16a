import type { Transport, Wire } from "../online-payment-with-vnpay";
import type { StoredOrder } from "../orders";

/**
 * UI-STAGE STAND-IN FOR THE SHOP, deleted in the backend step. It answers the two
 * calls the result page makes (GET /v1/orders/{id} and POST /v1/orders/{id}/payment)
 * for eight demo orders, so every state of the page can be reached by address:
 *
 *   /payment/return?vnp_TxnRef=DOI-24818-1&vnp_ResponseCode=00   paid
 *   /payment/return?vnp_TxnRef=DOI-24819-1&vnp_ResponseCode=00   confirming, then paid after 6 s
 *   /payment/return?vnp_TxnRef=DOI-24820-1&vnp_ResponseCode=00   confirming, then timed out after 30 s
 *   /payment/return?vnp_TxnRef=DOI-24820-1&vnp_ResponseCode=24   not completed (Try again opens VNPay)
 *   /payment/return?vnp_TxnRef=DOI-24821-1&vnp_ResponseCode=00   cancelled
 *   /payment/return?vnp_TxnRef=DOI-24822-1&vnp_ResponseCode=00   cancelled (the payment arrived late)
 *   /payment/return?vnp_TxnRef=DOI-24823-1&vnp_ResponseCode=00   could not check (the shop answers 500)
 *   /payment/return?vnp_TxnRef=DOI-24824-1&vnp_ResponseCode=24   not completed, Try again answers 503
 *   /payment/return?vnp_TxnRef=DOI-24825-1&vnp_ResponseCode=24   not completed, Try again answers 409: given up
 *   any other reference                                          cannot match
 */
interface Demo {
  number: string;
  status: string;
  paid: boolean;
  /** Becomes paid this long after the page first asked: VNPay's notice arriving. */
  paidAfterMs?: number;
  readFails?: boolean;
  /** What "Try again" is answered with. */
  pay: 200 | 409 | 503;
}

const DEMO: Record<string, Demo> = {
  "doi-24818": { number: "DOI-24818", status: "confirmed", paid: true, pay: 409 },
  "doi-24819": { number: "DOI-24819", status: "placed", paid: false, paidAfterMs: 6000, pay: 200 },
  "doi-24820": { number: "DOI-24820", status: "placed", paid: false, pay: 200 },
  "doi-24821": { number: "DOI-24821", status: "cancelled", paid: false, pay: 409 },
  "doi-24822": { number: "DOI-24822", status: "cancelled", paid: true, pay: 409 },
  "doi-24823": { number: "DOI-24823", status: "placed", paid: false, readFails: true, pay: 503 },
  "doi-24824": { number: "DOI-24824", status: "placed", paid: false, pay: 503 },
  "doi-24825": { number: "DOI-24825", status: "placed", paid: false, pay: 409 },
};

const tokenOf = (id: string) => `demo-token-${id}`;

/** The tokens a browser that had placed these orders would hold. */
export const demoHeld: StoredOrder[] = Object.entries(DEMO).map(([id, d]) => ({
  id,
  number: d.number,
  placed_at: "2026-11-26T09:00:00+07:00",
  token: tokenOf(id),
}));

const firstAsked = new Map<string, number>();

const answer = (status: number, body: unknown): Promise<Wire> =>
  new Promise((resolve) => setTimeout(() => resolve({ status, body }), 400));

const notFound = () => answer(404, { error: "not_found", message: "Not found." });

export const transport: Transport = {
  read(id, token) {
    const d = DEMO[id];
    if (!d || token !== tokenOf(id)) return notFound();
    if (d.readFails) return answer(500, { error: "internal", message: "Something went wrong." });
    const since = firstAsked.get(id) ?? Date.now();
    firstAsked.set(id, since);
    const paid = d.paid || (d.paidAfterMs !== undefined && Date.now() - since >= d.paidAfterMs);
    return answer(200, { id, number: d.number, status: d.status, paid });
  },
  pay(id, token) {
    const d = DEMO[id];
    if (!d || token !== tokenOf(id)) return notFound();
    if (d.pay === 503) return answer(503, { error: "payment_unavailable", message: "Online payment is not available right now." });
    if (d.pay === 409) {
      if (!d.paid) d.status = "cancelled";
      return answer(409, { error: "conflict", message: "This order can no longer be paid." });
    }
    return answer(200, { payment_url: `https://sandbox.vnpayment.vn/paymentv2/vpcpay.html?vnp_TxnRef=${d.number}-2` });
  },
};
