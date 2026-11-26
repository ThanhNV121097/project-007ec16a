import { useEffect, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { Button, Notice, PageHeader, Panel } from "./admin/index";
import OrderDelete from "./OrderDelete";
import OrderSummary from "./OrderSummary";
import OrderTabs from "./OrderTabs";
import {
  FIELD, changes, saveOrder, useStatusLabels, valuesOf,
  type DealerOrder, type EditableKey, type OrderValues,
} from "../lib/dealer-handles-orders-status-courier-payment";

type Phase = "idle" | "saving" | "saved" | "unreachable" | "expired";

/** Which fields the shop refused, by the name and tab the dealer knows them by. */
function refused(errors: Record<string, string>): string {
  return Object.keys(errors)
    .map((key) => (key in FIELD ? `${FIELD[key as EditableKey].label} (${FIELD[key as EditableKey].tab})` : key))
    .join(", ");
}

/** What the last save came to, said where it happened. */
function Said({ phase, errors, onClose }: { phase: Phase; errors: Record<string, string>; onClose: () => void }) {
  if (phase === "saving") return <Notice tone="info">Saving…</Notice>;
  if (phase === "saved") return <Notice tone="done" onClose={onClose}>Saved.</Notice>;
  if (phase === "unreachable") return <Notice tone="alert">We could not save. Please try again.</Notice>;
  if (phase === "expired") {
    return <Notice tone="alert">Your session ended. <Link to="/admin/sign-in" className="underline">Sign in again.</Link></Notice>;
  }
  if (Object.keys(errors).length > 0) return <Notice tone="alert">{`Nothing was saved. Check ${refused(errors)}.`}</Notice>;
  return null;
}

/**
 * One order, open, for the shop's team: the header with Delete and Save
 * changes, what the last save came to, the picture and sentence, and the tabs.
 *
 * It takes the order as a prop (the first paint) and hands the shop's answer
 * back through `onSaved`. A save carries only the editable fields that
 * changed; a refused one keeps every typed value and names its field.
 */
export default function OrderRecord({ order, title, paymentName, deliveryName, backTo, onSaved }: {
  order: DealerOrder;
  /** The heading the page shows for the order, which the delete question repeats. */
  title: string;
  paymentName: string;
  deliveryName: string;
  backTo: string;
  onSaved: (order: DealerOrder) => void;
}) {
  const navigate = useNavigate();
  const labels = useStatusLabels();
  const [values, setValues] = useState<OrderValues>(() => valuesOf(order));
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [phase, setPhase] = useState<Phase>("idle");
  const [confirming, setConfirming] = useState(false);

  // The order changes when a save is answered; the form follows it.
  useEffect(() => {
    setValues(valuesOf(order));
  }, [order]);

  async function save() {
    setPhase("saving");
    setErrors({});
    const result = await saveOrder(order.id, changes(order, values));
    if (result.ok) {
      onSaved(result.order);
      setPhase("saved");
    } else if (result.kind === "fields") {
      setErrors(result.errors);
      setPhase("idle");
    } else {
      setPhase(result.kind);
    }
  }

  const change = (key: EditableKey, value: string) => {
    setValues((cur) => ({ ...cur, [key]: value }));
    setErrors((cur) => {
      const next = { ...cur };
      delete next[key];
      return next;
    });
    setPhase((cur) => (cur === "saved" ? "idle" : cur));
  };

  return (
    <>
      <PageHeader
        title={title}
        actions={
          <>
            <Button kind="quiet" icon="remove" onClick={() => setConfirming(true)}>Delete</Button>
            <Button kind="primary" icon="check" disabled={phase === "saving"} onClick={() => void save()}>Save changes</Button>
          </>
        }
      />
      <Said phase={phase} errors={errors} onClose={() => setPhase("idle")} />
      <OrderSummary order={order} />
      <Panel>
        <OrderTabs order={order} values={values} errors={errors} labels={labels}
          paymentName={paymentName} deliveryName={deliveryName} onChange={change} />
      </Panel>
      <OrderDelete open={confirming} id={order.id} title={title} onClose={() => setConfirming(false)}
        onDeleted={() => navigate(backTo)} />
    </>
  );
}
