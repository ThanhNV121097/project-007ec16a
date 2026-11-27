/**
 * ONE RECORD, OPEN — its header, its tabs and what saving it does.
 *
 * The record arrives from the shop once and this form owns it from then on:
 * `base` is the record as the shop last said it, `values` what the person has
 * typed. A save sends only what changed, and what comes back — the shop's own
 * answer, with a name it trimmed and the lists it derived — replaces both, so
 * the page never shows what was typed as if it were stored.
 *
 * A refusal keeps every typed value and puts the shop's sentence beside its
 * field (the tab holding it opens); the shop being out of reach, or the
 * session ended, is a note above the form. On a saved shoe each size's count
 * saves by itself (StockCounts) and the lists and total are redrawn from the
 * shop's answer to it.
 */
import { useMemo, useState, type ReactNode } from "react";
import { useNavigate } from "react-router-dom";
import type { Row } from "../../data";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "../ui/tabs";
import { Breadcrumb, Button, Notice, PageHeader, Panel } from "./index";
import DeleteQuestion from "./DeleteQuestion";
import { Form } from "./Form";
import { missing, titleOf, type FieldSpec } from "./fields";
import { bodyOf, countsOf, stockErrors, valuesOf, type Values } from "./payload";
import SaveNote from "./SaveNote";
import type { StockLink } from "./StockCounts";
import { deleteRow, saveRow } from "./store";
import { useAction, useWorking } from "./useSave";

/** The tabs the fields ask for, in the order they first appear. */
function areasOf(specs: readonly FieldSpec[]): string[] {
  const areas: string[] = [];
  for (const f of specs) if (f.section && !areas.includes(f.section)) areas.push(f.section);
  return areas;
}

/** The first tab that holds a field the shop refused, so the sentence is not hidden behind another tab. */
function areaWithError(specs: readonly FieldSpec[], errors: Record<string, string>): string | undefined {
  const keys = Object.keys(errors);
  return specs.find((f) => f.section && keys.some((k) => k === f.key || (f.type === "sizes" && k.startsWith("pairs"))))?.section;
}

export default function RecordForm({ collection, record, specs, title, backTo, remove, children }: {
  collection: string;
  record: Row;
  specs: readonly FieldSpec[];
  /** The AREA's name — "Shoes" — for the crumb. */
  title: string;
  backTo?: string;
  remove: boolean;
  children?: ReactNode;
}) {
  const navigate = useNavigate();
  const [base, setBase] = useState<Row>(record);
  const [values, setValues] = useState<Values>(() => valuesOf(record));
  const [saved, setSaved] = useState(false);
  const [confirming, setConfirming] = useState(false);
  const areas = useMemo(() => areasOf(specs), [specs]);
  const [tab, setTab] = useState(areas[0] ?? "");
  const placed = useMemo(() => new Set(specs.map((f) => f.key)), [specs]);
  const act = useAction();
  const files = useWorking();

  const openTab = (errors: Record<string, string>) => {
    const area = areaWithError(specs, errors);
    if (area) setTab(area);
  };

  async function save() {
    setSaved(false);
    const bad = { ...missing(specs, values), ...stockErrors(values, base) };
    if (Object.keys(bad).length > 0) {
      act.refuse(bad);
      openTab(bad);
      return;
    }
    const body = bodyOf(specs, values, base);
    if (Object.keys(body).length === 0) {
      setSaved(true);
      return;
    }
    const out = await act.run(() => saveRow(collection, { ...body, id: base.id }));
    if (out?.ok) {
      setBase(out.value);
      setValues(valuesOf(out.value));
      setSaved(true);
    } else if (out?.kind === "fields") openTab(out.errors);
  }

  /** One size's count, as the shop now says it: its own, and the lists drawn from every count. */
  function counted(size: string, pairs: number, lists: Partial<Row>) {
    setBase((cur) => ({ ...cur, ...lists, pairs: { ...(cur.pairs as Record<string, number> | undefined), [size]: pairs } }));
    setValues((cur) => ({ ...cur, ...lists, pairs: { ...(cur.pairs as Record<string, string> | undefined), [size]: String(pairs) } }));
  }

  const stock: StockLink | undefined = specs.some((f) => f.type === "sizes")
    ? {
      productId: base.id,
      known: countsOf(base),
      onAnswer: (a) => counted(a.size, a.pairs, { sizes_low: a.sizes_low, sizes_out: a.sizes_out, stock: a.stock }),
      onStale: (size, fresh) => {
        const now = fresh ? countsOf(fresh)[size] : undefined;
        if (fresh && now !== undefined) counted(size, Number(now), { sizes_low: fresh.sizes_low, sizes_out: fresh.sizes_out, stock: fresh.stock });
      },
    }
    : undefined;

  const change = (key: string, v: unknown) => {
    setValues((cur) => ({ ...cur, [key]: v }));
    setSaved(false);
    act.clear(key);
  };
  const shown = { values, errors: act.errors, stock, onChange: change, onWorking: files.change };

  const form = areas.length > 0
    ? (
      <Tabs value={tab} onValueChange={setTab} className="gap-4">
        {/* Five tabs are wider than a 320px screen: the row scrolls, the page does not. */}
        <div className="max-w-full overflow-x-auto">
          <TabsList>
            {areas.map((area) => <TabsTrigger key={area} value={area}>{area}</TabsTrigger>)}
          </TabsList>
        </div>
        {areas.map((area) => (
          <TabsContent key={area} value={area}>
            <Form fields={specs.filter((f) => (f.section ?? "") === area)} {...shown} />
          </TabsContent>
        ))}
      </Tabs>
    )
    : <Form fields={specs} {...shown} />;

  return (
    <>
      {backTo && <Breadcrumb trail={[{ label: title, to: backTo }, { label: titleOf(base) }]} />}
      <PageHeader
        title={titleOf(base)}
        actions={
          <div aria-live="polite" className="contents">
            {remove && <Button kind="quiet" icon="remove" onClick={() => setConfirming(true)}>Delete</Button>}
            <Button kind="primary" icon="check" disabled={act.busy || files.working} onClick={() => void save()}>
              {act.busy ? "Saving…" : "Save changes"}
            </Button>
          </div>
        }
      />
      {saved && <Notice tone="done" onClose={() => setSaved(false)}>Saved.</Notice>}
      <SaveNote failure={act.failure} placed={placed} />
      <Panel>{form}</Panel>
      {children}
      {confirming && (
        <DeleteQuestion
          title={`Delete ${titleOf(base)}?`}
          onClose={() => setConfirming(false)}
          onDelete={() => deleteRow(collection, base.id)}
          onDeleted={() => {
            // Two calls, not navigate(backTo ?? -1): the router's delta overload
            // takes a number and its address overload a string, never a union.
            if (backTo) navigate(backTo);
            else navigate(-1);
          }}
        />
      )}
    </>
  );
}
