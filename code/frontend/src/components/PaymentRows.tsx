import { useContent } from "../editable";
import { cn } from "../lib/cn";
import type { PaymentOption } from "../lib/orders";
import Icon, { type IconName } from "./Icon";

function PayRow({ option, selected, unavailable, onChoose }: {
  option: PaymentOption;
  selected: boolean;
  unavailable: string;
  onChoose: (id: string) => void;
}) {
  const closed = !option.is_available;
  return (
    <div className="border-b border-line">
      <label data-testid={`pay-${option.id}`} data-selected={selected ? "true" : "false"} aria-disabled={closed ? "true" : undefined}
        className={cn(
          "flex gap-4 py-4 has-[:focus-visible]:outline has-[:focus-visible]:outline-2 has-[:focus-visible]:outline-offset-2 has-[:focus-visible]:outline-ink",
          closed ? "cursor-not-allowed opacity-60" : "cursor-pointer",
        )}>
        <input type="radio" name="payment" value={option.id} checked={selected} disabled={closed}
          onChange={() => onChoose(option.id)} className="sr-only" />
        <span aria-hidden="true" className={cn("mt-0.5 grid size-4 shrink-0 place-items-center rounded-full border", selected ? "border-ink" : "border-field")}>
          {selected && <span className="size-2 rounded-full bg-ink" />}
        </span>
        <Icon name={(option.icon || "card") as IconName} size={18} className="mt-0.5 shrink-0 text-ink" />
        <span className="min-w-0 flex-1">
          <span className="block text-sm text-ink">{option.name}</span>
          <span className="mt-1 block text-sm leading-relaxed text-ink-soft">{option.note}</span>
        </span>
      </label>
      {closed && (
        <p role="alert" className="flex items-center gap-2 pb-4 text-xs text-danger">
          <Icon name="alert" size={14} className="shrink-0" />{unavailable}
        </p>
      )}
    </div>
  );
}

/**
 * CHECKOUT'S STEP 03: one radio row (design system §2.1.17 — dot, icon, name
 * over note) for each way to pay the shop lists, in its order. One is chosen at
 * a time and the whole row is the click target; the real radio is there for the
 * keyboard (arrow keys move between the open rows). A way the shop lists but
 * cannot complete is drawn with the alert line under it and cannot be chosen.
 * The page names no way: the rows are whatever `options` holds.
 */
export default function PaymentRows({ options, chosen, onChoose }: {
  options: PaymentOption[];
  chosen: string;
  onChoose: (id: string) => void;
}) {
  const label = useContent("pages.checkout.step_payment");
  const unavailable = useContent("pages.checkout.unavailable");
  return (
    <div role="radiogroup" aria-label={label} className="border-t border-line">
      {options.map((o) => <PayRow key={o.id} option={o} selected={o.id === chosen} unavailable={unavailable} onChoose={onChoose} />)}
    </div>
  );
}
