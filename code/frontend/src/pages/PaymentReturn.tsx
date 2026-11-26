import { useSearchParams } from "react-router-dom";
import PaymentResult from "../components/PaymentResult";
import { usePaymentReturn } from "../lib/online-payment-with-vnpay";
import { usePageMeta } from "../seo";

/** What the page says for one address: a new address, a new start. */
function Outcome({ reference, code }: { reference: string | null; code: string | null }) {
  const r = usePaymentReturn(reference, code);
  return <PaymentResult state={r.state} order={r.order} trying={r.trying} tryFailed={r.tryFailed} onTryAgain={r.again} />;
}

/**
 * WHERE VNPAY SENDS THE CUSTOMER BACK (/payment/return). The address carries the
 * order's reference and VNPay's response code and is believed about neither
 * money nor status: the page asks the shop what the order is and says it in
 * plain words — paid, still confirming, not completed, cancelled — with the
 * way on from each (lib/online-payment-with-vnpay.ts).
 */
export default function PaymentReturn() {
  usePageMeta("pages.payment_return.title", "pages.payment_return.description");
  const [params] = useSearchParams();
  const reference = params.get("vnp_TxnRef");
  const code = params.get("vnp_ResponseCode");
  return <Outcome key={`${reference ?? ""}|${code ?? ""}`} reference={reference} code={code} />;
}
