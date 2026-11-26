import { useRef, useState, type FormEvent } from "react";
import { Link } from "react-router-dom";
import { W } from "../../lib/account-words";
import { addOrder, refreshMe, type OrderRow, type Outcome } from "../../lib/customer-accounts-sign-up-sign-in-my-orders";
import AccountBlock from "./AccountBlock";
import Field, { ErrorLine } from "./Field";
import { OUTLINE } from "./styles";

type Phase = "idle" | "adding" | "added" | "notfound" | "limited" | "failed";

function phaseFor(out: Outcome<OrderRow>): Phase {
  switch (out.kind) {
    case "ok": return "added";
    case "notfound": return "notfound";
    case "limited": return "limited";
    default: return "failed";
  }
}

/** A refusal sentence cut where "message us" is a link to the contact page. */
function Sentence({ parts }: { parts: readonly [string, string, string] }) {
  return (
    <ErrorLine>
      {parts[0]}<Link to="/contact" className="underline underline-offset-4">{parts[1]}</Link>{parts[2]}
    </ErrorLine>
  );
}

/**
 * ADD AN ORDER: a guest order, by its number and the phone it was placed
 * with, joins this account. A wrong pair, an unknown number and an order that
 * belongs to somebody else all read the same, so the page never says which.
 * The fields are sent as typed: the shop reads the number and phone the way
 * the order lookup does, and an empty phone is simply "no match".
 */
export default function AddOrder({ onAdded }: { onAdded: () => void }) {
  const sending = useRef(false);
  const [number, setNumber] = useState("");
  const [phone, setPhone] = useState("");
  const [phase, setPhase] = useState<Phase>("idle");

  const typed = (set: (v: string) => void) => (e: { target: { value: string } }) => {
    set(e.target.value);
    setPhase((p) => (p === "adding" ? p : "idle"));
  };

  async function submit(e: FormEvent) {
    e.preventDefault();
    if (sending.current) return;
    sending.current = true;
    setPhase("adding");
    const out = await addOrder(number, phone);
    sending.current = false;
    if (out.kind === "unauthenticated" || out.kind === "forbidden") void refreshMe();
    if (out.kind === "ok") {
      setNumber("");
      setPhone("");
      onAdded();
    }
    setPhase(phaseFor(out));
  }

  return (
    <AccountBlock id="add-order" title={W.add.title}>
      <form noValidate onSubmit={(e) => void submit(e)} aria-labelledby="add-order-title" className="flex max-w-md flex-col gap-4">
        <Field id="add-order-number" label={W.add.number} placeholder={W.add.placeholder} autoComplete="off" value={number} onChange={typed(setNumber)} />
        <Field id="add-order-phone" label={W.add.phone} type="tel" inputMode="tel" autoComplete="tel" value={phone} onChange={typed(setPhone)} />
        {phase === "notfound" && <Sentence parts={W.add.notFound} />}
        {phase === "limited" && <Sentence parts={W.add.limited} />}
        {phase === "failed" && <ErrorLine>{W.add.failed}</ErrorLine>}
        <div className="flex flex-wrap items-center gap-x-4">
          <button type="submit" disabled={phase === "adding"} className={OUTLINE}>{W.add.submit}</button>
          <p role="status" className="text-sm text-ink">{phase === "added" ? W.add.added : ""}</p>
        </div>
      </form>
    </AccountBlock>
  );
}
