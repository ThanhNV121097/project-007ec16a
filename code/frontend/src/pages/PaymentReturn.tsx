import { useSearchParams } from "react-router-dom";
import PaymentResult from "../components/PaymentResult";
import { readReturn } from "../lib/more-ways-to-pay-vietqr-transfer-and-momo";
import { usePaymentReturn } from "../lib/online-payment-with-vnpay";
import { usePageMeta } from "../seo";

/** What the page says for one address: a new address, a new start. */
function Outcome({ reference, code }: { reference: string | null; code: string | null }) {
  const r = usePaymentReturn(reference, code);
  return <PaymentResult state={r.state} order={r.order} way={r.way} trying={r.trying} tryFailed={r.tryFailed} onTryAgain={r.again} />;
}

/**
 * WHERE VNPAY AND MOMO SEND THE CUSTOMER BACK (/payment/return). The address
 * carries the order's reference and the provider's result code, in VNPay's
 * names or MoMo's, and is believed about neither money nor status: the page
 * asks the shop what the order is and says it in plain words — paid, still
 * confirming, not completed, cancelled — with the way on from each. Whose name
 * the words carry is the shop's answer about the order, not the address's
 * (lib/online-payment-with-vnpay.ts).
 */
export default function PaymentReturn() {
  usePageMeta("pages.payment_return.title", "pages.payment_return.description");
  const [params] = useSearchParams();
  const { reference, code } = readReturn(params);
  return <Outcome key={`${reference ?? ""}|${code ?? ""}`} reference={reference} code={code} />;
}
