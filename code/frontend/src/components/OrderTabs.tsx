import type { ReactNode } from "react";
import Choice from "./admin/Choice";
import { toneOf } from "./admin/fields";
import { Badge, Field, Input, Sections, Textarea } from "./admin/index";
import OrderFacts from "./OrderFacts";
import {
  FIELD, isLocked, movesFrom, type DealerOrder, type EditableKey, type OrderValues,
} from "../lib/dealer-handles-orders-status-courier-payment";
import type { OrderStatus } from "../lib/shop";

interface Props {
  order: DealerOrder;
  values: OrderValues;
  errors: Record<string, string>;
  labels: Record<OrderStatus, string>;
  paymentName: string;
  deliveryName: string;
  onChange: (key: EditableKey, value: string) => void;
}

/** A value that cannot be edited, drawn as text under its label. */
function Plain({ label, value, wide }: { label: string; value: string; wide?: boolean }) {
  return (
    <div className={wide ? "grid gap-1 md:col-span-2" : "grid gap-1"}>
      <p className="text-sm font-medium text-ink">{label}</p>
      <p className="whitespace-pre-line break-words text-sm text-ink">{value === "" ? "—" : value}</p>
    </div>
  );
}

/** One of the dealer's text fields, or its value as text once the order is delivered or cancelled. */
function Typed({ k, p, hint, type, wide }: { k: EditableKey; p: Props; hint?: string; type?: string; wide?: boolean }) {
  if (isLocked(p.order.status)) return <Plain label={FIELD[k].label} value={p.order[k] ?? ""} wide={wide} />;
  const error = p.errors[k];
  return (
    <div className={wide ? "md:col-span-2" : undefined}>
      <Field label={FIELD[k].label} hint={hint} error={error}>
        <Input type={type ?? "text"} value={p.values[k]} aria-invalid={error ? true : undefined}
          onChange={(e) => p.onChange(k, e.target.value)} />
      </Field>
    </div>
  );
}

/** The note is the one field that stays open on every order. */
function Note({ p }: { p: Props }) {
  const error = p.errors.note;
  return (
    <div className="md:col-span-2">
      <Field label={FIELD.note.label} error={error}>
        <Textarea value={p.values.note} aria-invalid={error ? true : undefined}
          onChange={(e) => p.onChange("note", e.target.value)} />
      </Field>
    </div>
  );
}

/** Offers the status the order has and the moves the shop allows from it, and nothing else. */
function StatusControl({ p }: { p: Props }) {
  const { order, labels } = p;
  if (isLocked(order.status)) {
    return (
      <div className="grid justify-items-start gap-1">
        <p className="text-sm font-medium text-ink">{FIELD.status.label}</p>
        <Badge tone={toneOf(order.status)}>{labels[order.status]}</Badge>
      </div>
    );
  }
  const options = [order.status, ...movesFrom(order.status)].map((s) => ({ value: s, label: labels[s] }));
  return (
    <Field label={FIELD.status.label} error={p.errors.status}>
      <Choice value={p.values.status} onChange={(v) => p.onChange("status", v)} aria-label={FIELD.status.label}
        options={options} />
    </Field>
  );
}

const GRID = "grid gap-x-4 gap-y-5 md:grid-cols-2";

function Grid({ children }: { children: ReactNode }) {
  return <div className={GRID}>{children}</div>;
}

/** The three tabs of an order: Order, Delivery, Customer. */
export default function OrderTabs(p: Props) {
  return (
    <Sections
      tabs={[
        {
          label: "Order",
          content: (
            <div className="grid gap-6">
              <StatusControl p={p} />
              <OrderFacts order={p.order} paymentName={p.paymentName} />
            </div>
          ),
        },
        {
          label: "Delivery",
          content: (
            <Grid>
              <Plain label="Delivery option" value={p.deliveryName} wide />
              <Typed k="carrier" p={p} />
              <Typed k="tracking" p={p} />
              <Typed k="eta" p={p} hint="Year-month-day, for example 2026-11-26" />
            </Grid>
          ),
        },
        {
          label: "Customer",
          content: (
            <Grid>
              <Typed k="customer" p={p} />
              <Typed k="phone" p={p} type="tel" />
              <Typed k="email" p={p} type="email" />
              <Typed k="city" p={p} />
              <Typed k="address" p={p} wide />
              <Note p={p} />
            </Grid>
          ),
        },
      ]}
    />
  );
}
