/**
 * THE FORM OVER ONE RECORD, WITH A CONTROL THAT FITS EACH FIELD.
 *
 * The same form in the add dialog and on the record page, because they are
 * the same form: a CMS where adding asks for one set of fields and editing
 * another is a CMS nobody can trust.
 *
 * WHAT EACH KIND OF FIELD GETS, and each of these replaced a plain text box:
 *
 *   a choice      a list to choose from — never a box to type "live" into
 *   a picture     the picture, sent to the shop when chosen (PictureControl)
 *   a price       a number with its currency beside it
 *   a list        tags you add and remove — never "S, M, L, XL" as a sentence
 *   the sizes     tags, and under them a count for each size (StockCounts)
 *   yes or no     a switch with its own line of explanation
 *   another record  the other collection's names — never its ids
 *   what the shop works out  a value, never a box: last pairs, sold out, stock
 *
 * A FIELD'S REFUSAL is drawn under its control as an alert, and the control
 * is marked invalid and points at it.
 *
 * TWO COLUMNS WHERE THE FIELDS ARE SHORT. Long text, pictures and lists take
 * the full width.
 */
import { cn } from "../../lib/cn";
import { siteCurrency } from "../../lib/format";
import { useCollection } from "../../data";
import Choice from "./Choice";
import { errorId, invalid, type ControlProps } from "./control-props";
import { Field, Input, Textarea, Toggle } from "./index";
import { text, titleOf, type FieldSpec } from "./fields";
import { cleanSize, listOf, type Values } from "./payload";
import PictureControl from "./PictureControl";
import StockCounts, { type StockLink } from "./StockCounts";
import TagsControl from "./TagsControl";

/** Fields that need the whole row: paragraphs, a picture and its buttons, a row of tags. */
function wide(spec: FieldSpec): boolean {
  if (spec.readOnly) return false;
  return spec.type === "longtext" || spec.type === "image" || spec.type === "list" || spec.type === "sizes";
}

/** A field pointing at another collection: that collection's names. */
function RefControl({ spec, value, onChange }: ControlProps) {
  const { items } = useCollection(spec.type.slice("ref:".length), { pageSize: 200 });
  return (
    <Field label={spec.label} hint={spec.hint}>
      <Choice value={text(value)} onChange={onChange} aria-label={spec.label}
        none={spec.required ? undefined : "None"}
        options={items.map((r) => ({ value: r.id, label: titleOf(r) }))} />
    </Field>
  );
}

/** A number of money, with what it is a number OF beside it. */
function PriceControl({ spec, value, error, onChange }: ControlProps) {
  return (
    <Field label={spec.label} hint={spec.hint}>
      <span className="relative block">
        <Input type="number" inputMode="decimal" min={0} value={text(value)} className="pr-14 tabular-nums"
          {...invalid(spec.key, error)}
          onChange={(e) => onChange(e.target.value === "" ? "" : Number(e.target.value))} />
        <span className="pointer-events-none absolute inset-y-0 right-3 flex items-center text-xs font-normal text-ink-soft">
          {siteCurrency()}
        </span>
      </span>
    </Field>
  );
}

/** What the shop works out: drawn as a value that is redrawn from its answer, never as a box. */
function Derived({ spec, value }: { spec: FieldSpec; value: unknown }) {
  const shown = text(value);
  const id = `field-${spec.key}-label`;
  return (
    <div className="grid gap-1">
      <span id={id} className="text-sm font-medium leading-none text-ink">{spec.label}</span>
      <output aria-labelledby={id} className="text-sm tabular-nums text-ink">{shown === "" ? "None" : shown}</output>
      {spec.hint && <p className="text-xs text-ink-soft">{spec.hint}</p>}
    </div>
  );
}

/** One control, chosen by what the field is. */
function Control(props: ControlProps & { onWorking?: (key: string, working: boolean) => void }) {
  const { spec, value, error, onChange } = props;
  if (spec.type === "bool") {
    return <Toggle checked={Boolean(value)} onChange={onChange} label={spec.label} hint={spec.hint} />;
  }
  if (spec.type.startsWith("ref:")) return <RefControl {...props} />;
  if (spec.type === "image") return <PictureControl {...props} />;
  if (spec.type === "list") return <TagsControl {...props} />;
  if (spec.type === "price") return <PriceControl {...props} />;
  if (spec.options) {
    return (
      <Field label={spec.label} hint={spec.hint}>
        <Choice value={text(value)} onChange={onChange} aria-label={spec.label}
          options={spec.options.map(([v, label]) => ({ value: v, label }))} />
      </Field>
    );
  }
  if (spec.type === "longtext") {
    return (
      <Field label={spec.label} hint={spec.hint}>
        <Textarea value={text(value)} {...invalid(spec.key, error)} onChange={(e) => onChange(e.target.value)} />
      </Field>
    );
  }
  const numeric = spec.type === "number";
  return (
    <Field label={spec.label} hint={spec.hint}>
      <Input type={numeric ? "number" : spec.type === "url" ? "url" : "text"} value={text(value)}
        className={numeric ? "tabular-nums" : undefined} {...invalid(spec.key, error)}
        onChange={(e) => onChange(numeric && e.target.value !== "" ? Number(e.target.value) : e.target.value)} />
    </Field>
  );
}

/** The sizes, and a count for each: the tags add and take off a size, the counts say how many pairs. */
function SizesBlock({ spec, values, errors, onChange, stock }: {
  spec: FieldSpec;
  values: Values;
  errors: Record<string, string>;
  onChange: (key: string, value: unknown) => void;
  stock?: StockLink;
}) {
  const counts = (values.pairs ?? {}) as Record<string, string>;
  return (
    <div className="grid gap-4">
      <TagsControl spec={spec} value={values[spec.key]} error={errors[spec.key]} normalise={cleanSize}
        onChange={(v) => onChange(spec.key, v)} />
      <StockCounts sizes={listOf(values[spec.key])} counts={counts} errors={errors} link={stock}
        onCount={(size, typed) => onChange("pairs", { ...counts, [size]: typed })} />
    </div>
  );
}

export function Form({ fields, values, errors = {}, onChange, columns = 2, stock, onWorking }: {
  fields: readonly FieldSpec[];
  values: Values;
  /** The shop's sentences, by field — and by `pairs.<size>` for a count. */
  errors?: Record<string, string>;
  onChange: (key: string, value: unknown) => void;
  /** 1 for a narrow place — a dialog on a phone decides for itself anyway. */
  columns?: 1 | 2;
  /** On a saved shoe's page: what lets each size's count save by itself. */
  stock?: StockLink;
  /** A control that sends a file says so here, so Save can wait for it. */
  onWorking?: (key: string, working: boolean) => void;
}) {
  return (
    <div className={cn("grid gap-x-4 gap-y-5", columns === 2 && "md:grid-cols-2")}>
      {fields.map((f) => (
        <div key={f.key} className={cn(columns === 2 && wide(f) && "md:col-span-2")}>
          {f.readOnly
            ? <Derived spec={f} value={values[f.key]} />
            : f.type === "sizes"
              ? <SizesBlock spec={f} values={values} errors={errors} onChange={onChange} stock={stock} />
              : <Control spec={f} value={values[f.key]} error={errors[f.key]} onWorking={onWorking}
                onChange={(v) => onChange(f.key, v)} />}
          {errors[f.key] && (
            <p id={errorId(f.key)} role="alert" className="mt-2 text-xs text-danger">{errors[f.key]}</p>
          )}
        </div>
      ))}
    </div>
  );
}
