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
    <ol className="flex flex-col gap-4 sm:grid sm:grid-cols-5 sm:gap-0" aria-label={labels[status]}>
      {ORDER_STEPS.map((s, i) => {
        const done = i <= at;
        return (
          <li key={s} aria-current={i === at ? "step" : undefined} className="relative flex items-center gap-3 sm:flex-col sm:items-stretch sm:pr-2">
            <span className={`hidden h-px w-full sm:block ${i < at ? "bg-ink" : "bg-line"}`} />
            <span className={`size-2.5 shrink-0 rounded-full sm:absolute sm:-top-1 sm:left-0 ${done ? "bg-ink" : "border border-field bg-ground"} ${i === at ? "outline outline-1 outline-offset-2 outline-ink" : ""}`} />
            <span className={`text-[0.6875rem] uppercase tracking-[0.1em] sm:text-xs ${done ? "text-ink" : "text-ink-soft"}`}>{labels[s]}</span>
          </li>
        );
      })}
    </ol>
  );
}
