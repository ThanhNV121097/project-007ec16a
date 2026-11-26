import { useContent } from "../editable";

/**
 * THE WORDS THE TWO NEW WAYS TO PAY ADD to the ones the design already had.
 * Each is a content key the owner edits like the rest of the site; until the
 * key is in content.json the sentence here is what shows, so a block is never
 * drawn with a label that says nothing.
 *
 *   pages.order.transfer.heading   .holder   .bank   .account   .amount   .note
 *   pages.order.transfer.line      .failed   .qr
 *   pages.payment_return.momo.timed_out.heading   .line
 */
const TRANSFER = {
  heading: "Pay by bank transfer",
  holder: "Account holder",
  bank: "Bank",
  account: "Account number",
  amount: "Amount",
  note: "Transfer note",
  line: "Scan this QR code with any banking app. We send the pair as soon as the money lands.",
  failed: "We could not load the bank transfer details. Please refresh the page.",
  qr: "QR code to pay order {number} by bank transfer",
} as const;

export function useTransferWords(): Record<keyof typeof TRANSFER, string> {
  const key = "pages.order.transfer";
  return {
    heading: useContent(`${key}.heading`) || TRANSFER.heading,
    holder: useContent(`${key}.holder`) || TRANSFER.holder,
    bank: useContent(`${key}.bank`) || TRANSFER.bank,
    account: useContent(`${key}.account`) || TRANSFER.account,
    amount: useContent(`${key}.amount`) || TRANSFER.amount,
    note: useContent(`${key}.note`) || TRANSFER.note,
    line: useContent(`${key}.line`) || TRANSFER.line,
    failed: useContent(`${key}.failed`) || TRANSFER.failed,
    qr: useContent(`${key}.qr`) || TRANSFER.qr,
  };
}

/** The result page's two sentences that name the provider, for an order that pays with MoMo. */
const MOMO_TIMED_OUT = {
  heading: "We have not heard from MoMo yet.",
  line: "If money left your account it will show on your order once MoMo confirms it. Please do not pay again.",
} as const;

export const MOMO_TIMED_OUT_KEY = "pages.payment_return.momo.timed_out";

export function useMomoTimedOutWords(): { heading: string; line: string } {
  return {
    heading: useContent(`${MOMO_TIMED_OUT_KEY}.heading`) || MOMO_TIMED_OUT.heading,
    line: useContent(`${MOMO_TIMED_OUT_KEY}.line`) || MOMO_TIMED_OUT.line,
  };
}
