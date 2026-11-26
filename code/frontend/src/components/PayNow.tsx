import { useEffect, useRef, useState } from "react";
<<<<<<< refs/heads/feature/order-page-and-pay-now-be
import { useContent } from "../editable";
import { openPayment, requestPayment } from "../lib/order-page-and-pay-now";
=======
import { openPayment, requestPayment } from "../lib/order-page-and-pay-now";
import { useOrderWords } from "../lib/order-words";
>>>>>>> refs/heads/main
import Icon from "./Icon";

type Phase = "idle" | "waiting" | "failed";

/**
 * "PAY NOW": the page's one commit action, a filled pill (design system
 * §2.1.6). Pressing it asks the shop for a payment address once and sends the
 * browser there; while the answer is awaited the button is disabled and says
 * so. Anything but an address — a refusal, an outage, a network failure — is
 * one line under the button in alert style, and the button works again.
 */
export default function PayNow({ orderId }: { orderId: string }) {
<<<<<<< refs/heads/feature/order-page-and-pay-now-be
  const pay = useContent("pages.order.pay_now");
  const paying = useContent("pages.order.paying");
  const failed = useContent("pages.order.pay_failed");
=======
  const words = useOrderWords();
>>>>>>> refs/heads/main
  const [phase, setPhase] = useState<Phase>("idle");
  const sending = useRef(false);
  const mounted = useRef(true);
  const button = useRef<HTMLButtonElement>(null);

  useEffect(() => {
    mounted.current = true;
    // Back from the payment page, the browser may restore this page as it was left: waiting.
    const restored = (e: PageTransitionEvent) => {
      if (!e.persisted) return;
      sending.current = false;
      setPhase("idle");
    };
    window.addEventListener("pageshow", restored);
    return () => {
      mounted.current = false;
      window.removeEventListener("pageshow", restored);
    };
  }, []);

  const press = async () => {
    if (sending.current) return;
    sending.current = true;
    setPhase("waiting");
    const result = await requestPayment(orderId);
    if (!mounted.current) return;
    if (result.ok) {
      openPayment(result.url);
      return;
    }
    sending.current = false;
    setPhase("failed");
    // A disabled button drops the keyboard's place: put it back.
    window.setTimeout(() => button.current?.focus(), 0);
  };

  return (
    <div>
      <button ref={button} type="button" data-testid="pay-now" disabled={phase === "waiting"} onClick={press}
        className="flex w-full items-center justify-center gap-2 rounded-[var(--radius-pill)] bg-ink px-6 py-3.5 text-xs font-medium uppercase tracking-[0.14em] text-ground hover:opacity-85 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ink disabled:opacity-60">
<<<<<<< refs/heads/feature/order-page-and-pay-now-be
        <Icon name="lock" size={16} />{phase === "waiting" ? paying : pay}
      </button>
      {phase === "failed" && (
        <p role="alert" data-testid="pay-now-error" className="mt-3 flex items-start gap-2 text-xs text-danger">
          <Icon name="alert" size={14} className="mt-0.5 shrink-0" />{failed}
=======
        <Icon name="lock" size={16} />{phase === "waiting" ? words.paying : words.pay}
      </button>
      {phase === "failed" && (
        <p role="alert" data-testid="pay-now-error" className="mt-3 flex items-start gap-2 text-xs text-danger">
          <Icon name="alert" size={14} className="mt-0.5 shrink-0" />{words.payFailed}
>>>>>>> refs/heads/main
        </p>
      )}
    </div>
  );
}
