import { useMemo, useState } from "react";
import type { Row } from "../../data";
import { Button, FormDialog } from "./index";
import { Form } from "./Form";
import { blank, missing, titleOf, type FieldSpec } from "./fields";
import { bodyOf, stockErrors, valuesOf, type Values } from "./payload";
import SaveNote from "./SaveNote";
import { saveRow } from "./store";
import { useAction, useWorking } from "./useSave";

/**
 * The add-and-edit dialog. Its own component so that its state — what has
 * been typed, what was refused — belongs to ONE record and is thrown away
 * with it.
 *
 * Saving asks the shop and waits: the button says "Saving…" and does nothing
 * more when pressed again. When the shop refuses, the dialog stays open with
 * everything that was typed, each refused field has its sentence beside it,
 * and any other refusal is the note above the form. Only an accepted save
 * closes it, and the list is then read again from the shop.
 */
export default function RecordDialog({ collection, specs, record, heading, onClose, onSaved }: {
  collection: string;
  specs: readonly FieldSpec[];
  /** The row being edited, a row with no id to add one, or null when shut. */
  record: Row | null;
  heading: string;
  onClose: () => void;
  onSaved: () => void;
}) {
  const adding = !record?.id;
  const existing = adding ? undefined : (record ?? undefined);
  const [values, setValues] = useState<Values>(() => (existing ? valuesOf(existing) : blank(specs)));
  const act = useAction();
  const files = useWorking();
  const roomy = specs.length > 5;
  const placed = useMemo(() => new Set(specs.map((f) => f.key)), [specs]);
  const close = () => { if (!act.busy) onClose(); };

  async function save() {
    const bad = { ...missing(specs, values), ...stockErrors(values, existing) };
    if (Object.keys(bad).length > 0) {
      act.refuse(bad);
      return;
    }
    const body = bodyOf(specs, values, existing);
    if (existing && Object.keys(body).length === 0) {
      onClose();
      return;
    }
    const out = await act.run(() => saveRow(collection, existing ? { ...body, id: existing.id } : body));
    if (out?.ok) onSaved();
  }

  return (
    <FormDialog
      open={record !== null}
      onClose={close}
      wide={roomy}
      title={adding ? `Add to ${heading.toLowerCase()}` : `Edit ${titleOf(existing)}`}
      description={adding ? undefined : "Changes show on the site as soon as they are saved."}
      footer={
        <div aria-live="polite" className="contents">
          <Button onClick={close} disabled={act.busy}>Cancel</Button>
          <Button kind="primary" disabled={act.busy || files.working} onClick={() => void save()}>
            {act.busy ? "Saving…" : adding ? "Add" : "Save changes"}
          </Button>
        </div>
      }
    >
      <div className="grid gap-4">
        <SaveNote failure={act.failure} placed={placed} />
        <Form fields={specs} values={values} errors={act.errors} columns={roomy ? 2 : 1}
          onWorking={files.change}
          onChange={(key, v) => { setValues((cur) => ({ ...cur, [key]: v })); act.clear(key); }} />
      </div>
    </FormDialog>
  );
}
