import Icon, { type IconName } from "./Icon";

/**
 * A row that is the only one there is: drawn selected, as the design draws a
 * chosen option, but with no control, because there is nothing to choose.
 * Checkout's step 02 (free delivery) and step 03 (VNPay) are each one of these.
 */
export default function CheckoutRow({ testid, icon, title, note, aside }: { testid: string; icon: IconName; title: string; note?: string; aside?: string }) {
  return (
    <div data-testid={testid} className="flex gap-4 border-b border-line py-4">
      <span aria-hidden="true" className="mt-0.5 grid size-4 shrink-0 place-items-center rounded-full border border-ink"><span className="size-2 rounded-full bg-ink" /></span>
      <Icon name={icon} size={18} className="mt-0.5 shrink-0 text-ink" />
      <span className="min-w-0 flex-1">
        <span className="block text-sm text-ink">{title}</span>
        <span className="mt-1 block text-sm leading-relaxed text-ink-soft">{note}</span>
      </span>
      {aside && <span className="shrink-0 text-sm tabular-nums text-ink">{aside}</span>}
    </div>
  );
}
