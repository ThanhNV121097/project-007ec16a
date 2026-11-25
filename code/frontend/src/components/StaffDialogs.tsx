import { useId, useRef, useState, type FormEvent } from "react";
import { useContent } from "../editable";
import { fill } from "../lib/shop";
import type { StaffAccount, StaffField, StaffInput, StaffPatch, StaffResult } from "../lib/staff";
import { Button, Field, FormDialog, Input } from "./admin/index";
import {
  AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription,
  AlertDialogFooter, AlertDialogHeader, AlertDialogTitle,
} from "./ui/alert-dialog";

/**
 * What is open over the staff table: the add form, the reset form, or one of
 * the two questions. One at a time, and each is its own component so what was
 * typed, what was refused and whether a save is in flight belong to that
 * dialog and are thrown away with it.
 */
export type StaffDialog =
  | { kind: "add" }
  | { kind: "reset" | "off" | "remove"; account: StaffAccount };

export interface StaffActions {
  add: (input: StaffInput) => Promise<StaffResult>;
  update: (id: string, patch: StaffPatch) => Promise<StaffResult>;
  remove: (id: string) => Promise<StaffResult>;
}

interface Problem {
  fields: Partial<Record<StaffField, string>>;
  message: string;
}

const NONE: Problem = { fields: {}, message: "" };

function describe(result: Exclude<StaffResult, { ok: true }>, couldNotSave: string): Problem {
  if (result.kind === "fields") return { fields: result.errors, message: "" };
  if (result.kind === "refused") return { fields: {}, message: result.message };
  return { fields: {}, message: couldNotSave };
}

/**
 * One save at a time. A second press while the first is in flight does
 * nothing (a ref, so two presses in one frame still make one account), the
 * dialog cannot be closed under it, and what the shop said is kept for the
 * dialog to show.
 */
function useAttempt(onDone: () => void, onClose: () => void) {
  const [saving, setSaving] = useState(false);
  const [problem, setProblem] = useState<Problem>(NONE);
  const busy = useRef(false);
  const couldNotSave = useContent("pages.admin.staff.couldNotSave");

  const run = (send: () => Promise<StaffResult>) => {
    if (busy.current) return;
    busy.current = true;
    setSaving(true);
    setProblem(NONE);
    void send().then((result) => {
      busy.current = false;
      setSaving(false);
      if (result.ok) onDone();
      else setProblem(describe(result, couldNotSave));
    });
  };
  const close = () => {
    if (!busy.current) onClose();
  };
  return { saving, problem, run, close };
}

/** The shop's sentence for what could not be done, announced as it appears. */
function Said({ children }: { children: string }) {
  return <p role="alert" className="rounded-md border border-danger/40 px-3 py-2 text-sm text-danger">{children}</p>;
}

/** A labelled box whose refusal sits beside it, marked invalid and announced. */
function StaffField({ label, hint, error, ...input }: {
  label: string;
  hint?: string;
  error?: string;
} & React.InputHTMLAttributes<HTMLInputElement>) {
  const id = useId();
  return (
    <div className="grid gap-2">
      <Field label={label} hint={hint}>
        <Input {...input} aria-invalid={error ? true : undefined} aria-describedby={error ? id : undefined} />
      </Field>
      {error && <p id={id} role="alert" className="text-xs text-danger">{error}</p>}
    </div>
  );
}

function AddDialog({ add, onClose }: { add: StaffActions["add"]; onClose: () => void }) {
  const [values, setValues] = useState<StaffInput>({ name: "", email: "", password: "" });
  const attempt = useAttempt(onClose, onClose);
  const cancel = useContent("pages.admin.staff.dialog.add.cancel");
  const submit = useContent("pages.admin.staff.dialog.add.submit");
  const saving = useContent("pages.admin.staff.saving");
  const { fields, message } = attempt.problem;
  const change = (key: keyof StaffInput) => (e: React.ChangeEvent<HTMLInputElement>) =>
    setValues((cur) => ({ ...cur, [key]: e.target.value }));
  const send = (e: FormEvent) => {
    e.preventDefault();
    attempt.run(() => add(values));
  };

  return (
    <FormDialog
      open
      onClose={attempt.close}
      title={useContent("pages.admin.staff.dialog.add.title")}
      footer={
        <>
          <Button type="button" onClick={attempt.close}>{cancel}</Button>
          <Button kind="primary" type="submit" form="staff-add" disabled={attempt.saving}>
            {attempt.saving ? saving : submit}
          </Button>
        </>
      }
    >
      <form id="staff-add" onSubmit={send} noValidate className="grid gap-5">
        {message && <Said>{message}</Said>}
        <StaffField label={useContent("pages.admin.staff.dialog.add.name")} name="name" autoComplete="off"
          value={values.name} onChange={change("name")} error={fields.name} />
        <StaffField label={useContent("pages.admin.staff.dialog.add.email")} name="email" type="email" inputMode="email"
          autoComplete="off" value={values.email} onChange={change("email")} error={fields.email} />
        <StaffField label={useContent("pages.admin.staff.dialog.add.password")} name="password" type="password"
          autoComplete="new-password" hint={useContent("pages.admin.staff.dialog.add.hint")}
          value={values.password} onChange={change("password")} error={fields.password} />
      </form>
    </FormDialog>
  );
}

function ResetDialog({ account, update, onClose, onSaved }: {
  account: StaffAccount;
  update: StaffActions["update"];
  onClose: () => void;
  onSaved: () => void;
}) {
  const [password, setPassword] = useState("");
  const attempt = useAttempt(() => { onClose(); onSaved(); }, onClose);
  const cancel = useContent("pages.admin.staff.dialog.reset.cancel");
  const submit = useContent("pages.admin.staff.dialog.reset.submit");
  const saving = useContent("pages.admin.staff.saving");
  const title = fill(useContent("pages.admin.staff.dialog.reset.title"), { name: account.name });
  const send = (e: FormEvent) => {
    e.preventDefault();
    attempt.run(() => update(account.id, { password }));
  };

  return (
    <FormDialog
      open
      onClose={attempt.close}
      title={title}
      footer={
        <>
          <Button type="button" onClick={attempt.close}>{cancel}</Button>
          <Button kind="primary" type="submit" form="staff-reset" disabled={attempt.saving}>
            {attempt.saving ? saving : submit}
          </Button>
        </>
      }
    >
      <form id="staff-reset" onSubmit={send} noValidate className="grid gap-5">
        {attempt.problem.message && <Said>{attempt.problem.message}</Said>}
        <StaffField label={useContent("pages.admin.staff.dialog.reset.password")} name="password" type="password"
          autoComplete="new-password" value={password} onChange={(e) => setPassword(e.target.value)}
          error={attempt.problem.fields.password} />
      </form>
    </FormDialog>
  );
}

/**
 * A question before an account is switched off or removed. It is the same
 * alert dialog Confirm is made of, because Confirm closes the moment its
 * button is pressed and has nowhere to say that the shop refused: here the
 * question stays open, shows the shop's sentence, and the button reads
 * "Saving…" while the answer is awaited.
 */
function Question({ title, note, verb, danger, send, onClose }: {
  title: string;
  note: string;
  verb: string;
  danger?: boolean;
  send: () => Promise<StaffResult>;
  onClose: () => void;
}) {
  const attempt = useAttempt(onClose, onClose);
  const cancel = useContent("pages.admin.staff.dialog.add.cancel");
  const saving = useContent("pages.admin.staff.saving");
  return (
    <AlertDialog open onOpenChange={(next) => { if (!next) attempt.close(); }}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle className="font-body">{title}</AlertDialogTitle>
          <AlertDialogDescription>{note}</AlertDialogDescription>
        </AlertDialogHeader>
        {attempt.problem.message && <Said>{attempt.problem.message}</Said>}
        <AlertDialogFooter>
          <AlertDialogCancel>{cancel}</AlertDialogCancel>
          <AlertDialogAction
            variant={danger ? "destructive" : "default"}
            disabled={attempt.saving}
            onClick={(e) => { e.preventDefault(); attempt.run(send); }}
          >
            {attempt.saving ? saving : verb}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}

export default function StaffDialogs({ dialog, actions, onClose, onSaved }: {
  dialog: StaffDialog | null;
  actions: StaffActions;
  onClose: () => void;
  /** After a password reset: the page says "Saved." */
  onSaved: () => void;
}) {
  const off = useContent("pages.admin.staff.confirm.off.title");
  const offNote = useContent("pages.admin.staff.confirm.off.note");
  const offVerb = useContent("pages.admin.staff.confirm.off.verb");
  const gone = useContent("pages.admin.staff.confirm.remove.title");
  const goneNote = useContent("pages.admin.staff.confirm.remove.note");
  const goneVerb = useContent("pages.admin.staff.confirm.remove.verb");
  if (!dialog) return null;
  if (dialog.kind === "add") return <AddDialog add={actions.add} onClose={onClose} />;
  const { account } = dialog;
  if (dialog.kind === "reset") {
    return <ResetDialog key={account.id} account={account} update={actions.update} onClose={onClose} onSaved={onSaved} />;
  }
  if (dialog.kind === "off") {
    return (
      <Question key={account.id} title={fill(off, { name: account.name })} note={offNote} verb={offVerb}
        send={() => actions.update(account.id, { is_active: false })} onClose={onClose} />
    );
  }
  return (
    <Question key={account.id} title={fill(gone, { name: account.name })} note={goneNote} verb={goneVerb} danger
      send={() => actions.remove(account.id)} onClose={onClose} />
  );
}
