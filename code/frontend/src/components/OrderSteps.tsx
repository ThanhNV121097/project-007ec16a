import { useContent } from "../editable";
import { ORDER_STEPS, type OrderStatus } from "../lib/shop";
import Icon from "./Icon";

/**
 * WHERE AN ORDER IS, AS ONE LINE: the five steps on a hairline, the ones
 * done drawn in ink and the current one ringed. A cancelled order says so
 * instead of pretending to be on its way.
 */
export default function OrderSteps({ status }: { status: OrderStatus }) {
  const labels: Record<string, string> = {
    placed: useContent("orders.steps.placed"),
    confirmed: useContent("orders.steps.confirmed"),
    packed: useContent("orders.steps.packed"),
    shipped: useContent("orders.steps.shipped"),
    delivered: useContent("orders.steps.delivered"),
  };
  const cancelled = useContent("orders.cancelled");
  if (status === "cancelled") {
    return (
      <p className="flex items-center gap-2 border-y border-line py-4 text-sm text-ink">
        <Icon name="alert" className="shrink-0 text-danger" /> {cancelled}
      </p>
    );
  }
  const at = ORDER_STEPS.indexOf(status);
  return (
    <ol className="grid grid-cols-5" aria-label={labels[status]}>
      {ORDER_STEPS.map((s, i) => {
        const done = i <= at;
        return (
          <li key={s} aria-current={i === at ? "step" : undefined} className="relative flex flex-col gap-3 pr-2">
            <span className={`h-px w-full ${i < at ? "bg-ink" : "bg-line"}`} />
            <span className={`absolute -top-1 left-0 size-2.5 rounded-full ${done ? "bg-ink" : "border border-field bg-ground"} ${i === at ? "outline outline-1 outline-offset-2 outline-ink" : ""}`} />
            <span className={`text-[0.6875rem] uppercase tracking-[0.1em] sm:text-xs ${done ? "text-ink" : "text-ink-soft"}`}>{labels[s]}</span>
          </li>
        );
      })}
    </ol>
  );
}
