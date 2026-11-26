import { useContent } from "../editable";

/**
 * THE WORDS THE ORDER PAGE ADDS to the ones the design already had. Each is a
 * content key the owner edits like the rest of the site; until the key is in
 * content.json the sentence here is what shows, so a page is never drawn with
 * a button that says nothing.
 *
 *   pages.order.pay_now      pages.order.paying      pages.order.pay_failed
 *   pages.order.load_failed  pages.order.retry
 */
const WORDS = {
  pay: "Pay now",
  paying: "Opening the payment page…",
  payFailed: "We could not open the payment page. Please try again.",
  loadFailed: "We could not load your order. Please try again.",
  retry: "Try again",
} as const;

export function useOrderWords(): Record<keyof typeof WORDS, string> {
  const pay = useContent("pages.order.pay_now");
  const paying = useContent("pages.order.paying");
  const payFailed = useContent("pages.order.pay_failed");
  const loadFailed = useContent("pages.order.load_failed");
  const retry = useContent("pages.order.retry");
  return {
    pay: pay || WORDS.pay,
    paying: paying || WORDS.paying,
    payFailed: payFailed || WORDS.payFailed,
    loadFailed: loadFailed || WORDS.loadFailed,
    retry: retry || WORDS.retry,
  };
}
