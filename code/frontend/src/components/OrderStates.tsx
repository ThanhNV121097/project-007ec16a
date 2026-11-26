<<<<<<< refs/heads/feature/order-page-and-pay-now-be
import { T, useContent } from "../editable";
=======
import { T } from "../editable";
import { useOrderWords } from "../lib/order-words";
>>>>>>> refs/heads/main
import Icon from "./Icon";
import OutlinePill from "./OutlinePill";

/**
 * While the shop has not answered: a quiet block in the page's place. No
 * heading and no sentence, so nothing on it can be mistaken for an answer
 * — least of all "we can't find that order".
 */
export function OrderLoading() {
  return (
    <div aria-busy="true" data-testid="order-loading" className="min-h-96 py-16">
      <div className="h-3 w-40 bg-surface" />
      <div className="mt-6 h-16 w-full max-w-md bg-surface" />
      <div className="mt-6 h-3 w-56 bg-surface" />
    </div>
  );
}

/** The shop answered 404: no order, no token or another order's token — one view for all three. */
export function OrderMissing() {
  return (
    <section data-testid="order-missing" className="flex flex-col items-start gap-5 py-16">
      <T k="pages.order.missing" as="h1" className="max-w-[16ch] text-[length:var(--size-h2)] text-ink" />
      <T k="pages.order.missing_line" as="p" className="max-w-[44ch] text-sm text-ink-soft" />
      <OutlinePill to="/track"><T k="pages.order.track_cta" /></OutlinePill>
    </section>
  );
}

/** The shop could not be asked or failed: neither the order nor "not found", one line and a way to ask again. */
export function OrderLoadError({ onRetry }: { onRetry: () => void }) {
<<<<<<< refs/heads/feature/order-page-and-pay-now-be
  const failed = useContent("pages.order.load_failed");
  const retry = useContent("pages.order.retry");
  return (
    <section className="flex flex-col items-start gap-5 py-16">
      <p role="alert" data-testid="order-load-error" className="flex items-center gap-2 text-sm text-ink">
        <Icon name="alert" size={16} className="shrink-0 text-danger" />{failed}
      </p>
      <button type="button" onClick={onRetry}
        className="inline-flex items-center gap-2 rounded-[var(--radius-pill)] border border-ink px-4 py-2 text-[0.6875rem] font-medium uppercase tracking-[0.12em] text-ink transition-colors hover:bg-ink hover:text-ground focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ink">
        {retry}
=======
  const words = useOrderWords();
  return (
    <section className="flex flex-col items-start gap-5 py-16">
      <p role="alert" data-testid="order-load-error" className="flex items-center gap-2 text-sm text-ink">
        <Icon name="alert" size={16} className="shrink-0 text-danger" />{words.loadFailed}
      </p>
      <button type="button" onClick={onRetry}
        className="inline-flex items-center gap-2 rounded-[var(--radius-pill)] border border-ink px-4 py-2 text-[0.6875rem] font-medium uppercase tracking-[0.12em] text-ink transition-colors hover:bg-ink hover:text-ground focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ink">
        {words.retry}
>>>>>>> refs/heads/main
      </button>
    </section>
  );
}
