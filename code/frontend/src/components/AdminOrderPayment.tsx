import { useEffect, useRef, useState } from "react";
import { Link } from "react-router-dom";
import { BANK, markPaid } from "../lib/more-ways-to-pay-vietqr-transfer-and-momo";
import type { OrderStatus } from "../lib/shop";
import { Badge, Button } from "./admin/index";

/** The providers whose own transaction number the dealer reads beside the status. */
const PROVIDER: Record<string, string> = { vnpay: "VNPay", momo: "MoMo" };

type Phase = "idle" | "saving" | "saved" | "failed" | "expired";

/**
 * "PAYMENT STATUS" ON THE DEALER'S ORDER PAGE: Paid or Awaiting payment, the
 * provider's transaction number once it has confirmed (read-only), and — for an
 * unpaid bank-transfer order, whatever its status, and no other order — the one
 * button that marks it paid once the money has landed in the shop's account.
 *
 * The button sends `paid: true` and nothing else. While it waits it is disabled
 * and says so; a refusal or a lost connection says "We could not save" and
 * enables it again; a session that ended says so and links to sign-in. Marking
 * is the shop's decision: this block only shows what it answered, and hands the
 * order's new status to `onMarked` so the page can show "Confirmed".
 */
export default function AdminOrderPayment({ id, way, paid, status, transactionNo = "", onMarked }: {
  id: string;
  /** The order's way to pay: only a bank-transfer order is marked paid by hand. */
  way: string;
  paid: boolean;
  status: OrderStatus;
  transactionNo?: string;
  onMarked?: (next: { paid: true; status: OrderStatus }) => void;
}) {
  const [phase, setPhase] = useState<Phase>("idle");
  const [marked, setMarked] = useState(false);
  const box = useRef<HTMLDivElement>(null);
  const button = useRef<HTMLButtonElement>(null);
  const isPaid = paid || marked;
  const provider = PROVIDER[way];

  // The button leaves when the order is paid and is disabled while it waits: keep the keyboard's place.
  useEffect(() => {
    if (phase === "saved") box.current?.focus();
    if (phase === "failed") button.current?.focus();
  }, [phase]);

  const press = async () => {
    setPhase("saving");
    const result = await markPaid(id, status);
    if (!result.ok) {
      setPhase(result.kind);
      return;
    }
    setMarked(true);
    setPhase("saved");
    onMarked?.({ paid: true, status: result.status });
  };

  return (
    <div ref={box} tabIndex={-1} data-testid="admin-payment" className="grid gap-3 rounded-md border border-line bg-surface p-4 outline-none">
      <p className="text-xs text-ink-soft">Payment status</p>
      <div className="flex flex-wrap items-center gap-3">
        <Badge tone={isPaid ? "good" : "warn"}>{isPaid ? "Paid" : "Awaiting payment"}</Badge>
        {way === BANK && !isPaid && (
          <Button ref={button} data-testid="admin-mark-paid" disabled={phase === "saving"} onClick={() => void press()}>
            {phase === "saving" ? "Saving…" : "Mark as paid"}
          </Button>
        )}
      </div>
      {provider && transactionNo !== "" && <p className="text-sm text-ink">{`${provider} transaction ${transactionNo}`}</p>}
      {phase === "saved" && <p role="status" className="text-sm text-ink">Saved.</p>}
      {phase === "failed" && <p role="alert" className="text-sm text-danger">We could not save. Please try again.</p>}
      {phase === "expired" && (
        <p role="alert" className="text-sm text-danger">
          Your session ended. <Link to="/admin/sign-in" className="underline">Sign in again.</Link>
        </p>
      )}
    </div>
  );
}
