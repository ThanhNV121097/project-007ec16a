import { useMemo } from "react";
import { formatPrice } from "../lib/format";
import { qrRows, useTransfer, type TransferDetails } from "../lib/more-ways-to-pay-vietqr-transfer-and-momo";
import type { PublicOrder } from "../lib/order-page-and-pay-now";
import { storedOrders } from "../lib/orders";
import { useTransferWords } from "../lib/payment-words";
import { fill } from "../lib/shop";
import Icon from "./Icon";
import TransferQr from "./TransferQr";

/** Where the block sits in the order page's grid: a full row above the pairs, first on a phone too. */
const SLOT = "order-first min-w-0 lg:col-span-12";

function Row({ label, value, testid }: { label: string; value: string; testid: string }) {
  return (
    <div className="border-t border-line py-3 first:border-t-0 first:pt-0 sm:grid sm:grid-cols-[minmax(9rem,auto)_1fr] sm:gap-4">
      <dt className="text-[0.6875rem] font-medium uppercase tracking-[0.12em] text-ink-soft">{label}</dt>
      <dd data-testid={testid} className="mt-1 select-all wrap-anywhere text-base text-ink sm:mt-0">{value}</dd>
    </div>
  );
}

function Details({ order, details }: { order: Pick<PublicOrder, "number">; details: TransferDetails }) {
  const words = useTransferWords();
  const rows = useMemo(() => qrRows(details.qr), [details.qr]);
  return (
    <section data-testid="transfer-block" aria-labelledby="transfer-heading" className={`${SLOT} border border-line p-4 sm:p-6`}>
      <h2 id="transfer-heading" className="text-lg text-ink">{words.heading}</h2>
      <div className="mt-5 flex min-w-0 flex-col gap-6 lg:flex-row lg:gap-8">
        <div className="w-full max-w-64 shrink-0 lg:w-64">
          <TransferQr rows={rows} label={fill(words.qr, { number: order.number })} />
        </div>
        <div className="min-w-0 flex-1">
          <dl className="text-sm">
            <Row label={words.holder} value={details.holder} testid="transfer-holder" />
            <Row label={words.bank} value={details.bank} testid="transfer-bank" />
            <Row label={words.account} value={details.account_number} testid="transfer-account" />
            <Row label={words.amount} value={formatPrice(details.amount)} testid="transfer-amount" />
            <Row label={words.note} value={details.note} testid="transfer-note" />
          </dl>
          <p className="mt-4 max-w-[48ch] text-sm leading-relaxed text-ink-soft">{words.line}</p>
        </div>
      </div>
    </section>
  );
}

/**
 * "PAY BY BANK TRANSFER": for an unpaid bank-transfer order, the QR code and the
 * shop's account written out for typing by hand, in place of "Pay now".
 *
 * Three states beside the details, none of them the not-found view: while the
 * shop is asked, a quiet block of the QR's size (the rest of the page is already
 * drawn); when it cannot be asked, one line in alert style where the block
 * would be; and when the account is no longer set, nothing at all.
 */
export default function TransferBlock({ order }: { order: Pick<PublicOrder, "id" | "number" | "total"> }) {
  // The order's own token, kept by this browser when it placed the order; none held means the signed-in customer's session speaks.
  const read = useTransfer(order.id, storedOrders().find((o) => o.id === order.id)?.token);
  const words = useTransferWords();
  if (read.state === "gone") return null;
  if (read.state === "loading") {
    return (
      <div aria-busy="true" data-testid="transfer-loading" className={SLOT}>
        <div className="aspect-square w-full max-w-64 bg-surface" />
      </div>
    );
  }
  if (read.state === "failed") {
    return (
      <p role="alert" data-testid="transfer-failed" className={`${SLOT} flex items-start gap-2 text-sm text-danger`}>
        <Icon name="alert" size={16} className="mt-0.5" />{words.failed}
      </p>
    );
  }
  return <Details order={order} details={read.details} />;
}
