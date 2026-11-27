import { Button, Empty } from "./admin/index";

/**
 * What an order's page says while it has no order to draw: still asking the
 * shop, could not ask, or the shop answered that there is no such order.
 * "This record is not here" is only ever the shop's own answer, never a guess
 * made while the answer is on its way.
 */
export default function OrderState({ state, onRetry, onBack }: {
  state: "loading" | "failed" | "missing";
  onRetry: () => void;
  onBack: () => void;
}) {
  if (state === "loading") return <p role="status" className="text-sm text-ink-soft">Loading the order…</p>;
  if (state === "failed") {
    return (
      <Empty title="We could not load this order." note="Check the connection and try again."
        action={<Button onClick={onRetry}>Try again</Button>} />
    );
  }
  return (
    <Empty title="This record is not here" note="It may have been deleted since the link was made."
      action={<Button onClick={onBack}>Back to the list</Button>} />
  );
}
