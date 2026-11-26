import { T, useContent } from "../editable";
import type { PaymentState } from "../lib/online-payment-with-vnpay";
import { fill } from "../lib/shop";
import Icon from "./Icon";
import OutlinePill from "./OutlinePill";

const KEY = "pages.payment_return";

/** The filled pill (design system §2.1.6): the page's one commit action. */
const FILLED =
  "inline-flex items-center justify-center gap-2 rounded-[var(--radius-pill)] bg-ink px-6 py-3.5 text-xs font-medium uppercase tracking-[0.14em] text-ground hover:opacity-85 disabled:opacity-60";

const BIG = "max-w-[18ch] text-[length:var(--size-h1)] leading-[1.02] text-ink break-words";
const LONG = "max-w-[24ch] text-[length:var(--size-h2)] leading-[1.1] text-ink break-words";

interface Order {
  id: string;
  number: string;
}

export interface PaymentResultProps {
  /** `loading` until the shop has answered the first time. */
  state: PaymentState | "loading";
  order: Order | null;
  trying: boolean;
  tryFailed: boolean;
  onTryAgain: () => void;
}

const wordsOf = (state: PaymentState) => `${KEY}.${state.replace("-", "_")}`;

/** The one h1: what became of the payment. "Could not check" has no heading of its own, so its alert is the heading. */
function Heading({ state }: { state: PaymentState }) {
  if (state === "check-failed") {
    return (
      <h1 data-testid="payment-state" data-state={state} className={LONG}>
        <span role="alert" className="flex items-start gap-2 text-danger">
          <Icon name="alert" size={20} className="mt-1" />
          <T k={`${KEY}.check_failed.line`} />
        </span>
      </h1>
    );
  }
  const long = state === "no-match";
  return <T k={`${wordsOf(state)}.heading`} as="h1" data-testid="payment-state" data-state={state} className={long ? LONG : BIG} />;
}

/** The state line, with the order number where its words ask for one. */
function Line({ state, number }: { state: PaymentState; number: string }) {
  const key = `${wordsOf(state)}.line`;
  const text = useContent(key);
  return (
    <p data-testid="payment-line" data-edit={key} className="max-w-[44ch] text-sm leading-relaxed text-ink-soft">
      {fill(text ?? "", { number })}
    </p>
  );
}

function TryAgain({ trying, failed, onPress }: { trying: boolean; failed: boolean; onPress: () => void }) {
  return (
    <div className="flex flex-col items-start gap-3">
      <button type="button" data-testid="payment-try-again" disabled={trying} aria-busy={trying || undefined} onClick={onPress} className={FILLED}>
        <T k={`${KEY}.try_again`} />
      </button>
      {failed && (
        <p role="alert" className="flex items-center gap-2 text-xs text-danger">
          <Icon name="alert" size={14} />
          <T k={`${KEY}.try_again_error`} />
        </p>
      )}
    </div>
  );
}

function ViewOrder({ order }: { order: Order }) {
  return (
    <OutlinePill to={`/orders/${order.id}`} testid="payment-view-order">
      <T k={`${KEY}.view_order`} />
    </OutlinePill>
  );
}

/** What can be pressed: the way on from each state. */
function Actions({ state, order, trying, tryFailed, onTryAgain }: Omit<PaymentResultProps, "state"> & { state: PaymentState }) {
  if (!order) {
    return (
      <OutlinePill to="/track" testid="payment-track">
        <T k={`${KEY}.track`} />
      </OutlinePill>
    );
  }
  return (
    <div className="flex flex-wrap items-start gap-3">
      {state === "not-completed" && <TryAgain trying={trying} failed={tryFailed} onPress={onTryAgain} />}
      <ViewOrder order={order} />
      {state === "cancelled" && (
        <OutlinePill to="/shop" testid="payment-keep-shopping">
          <T k={`${KEY}.keep_shopping`} />
        </OutlinePill>
      )}
    </div>
  );
}

/**
 * WHAT BECAME OF THE PAYMENT, in plain words: one heading, one line, the way on.
 * Until the shop has answered there is only a quiet placeholder — never a state
 * guessed from the address. The line sits in a status region that is there from
 * the first paint, so the change from "confirming" to "paid" is announced.
 */
export default function PaymentResult({ state, order, trying, tryFailed, onTryAgain }: PaymentResultProps) {
  const loading = state === "loading";
  const known = loading ? null : state;
  return (
    <section className="flex flex-col items-start gap-5 py-16 lg:py-24">
      {known && <Heading state={known} />}
      <div role="status" aria-busy={loading || undefined} className="w-full max-w-[44ch]">
        {loading && (
          <div aria-hidden="true" className="flex flex-col gap-4">
            <span className="block h-10 w-3/4 bg-surface" />
            <span className="block h-4 w-full bg-surface" />
          </div>
        )}
        {known && known !== "check-failed" && <Line state={known} number={order?.number ?? ""} />}
      </div>
      {known && <Actions state={known} order={order} trying={trying} tryFailed={tryFailed} onTryAgain={onTryAgain} />}
    </section>
  );
}
