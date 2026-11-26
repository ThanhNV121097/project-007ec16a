import { useRef, useState } from "react";
import Icon from "../Icon";
import { text } from "./fields";
import { errorId, invalid, type ControlProps } from "./control-props";
import { Button, Field } from "./index";
import { FailureLine } from "./SaveNote";
import { uploadFile, type Failure } from "./store";

/**
 * A PICTURE IS CHOSEN BY LOOKING AT IT, AND SENT TO THE SHOP AT ONCE.
 *
 * The file goes to the shop the moment it is chosen and the control keeps the
 * address the shop answers with — never the file's own bytes as text. While it
 * is on its way the control says so and tells the form (`onWorking`), which
 * makes Save wait; a refusal (a GIF, a file over 5 MB, no answer) shows beside
 * the control and every other typed value stays.
 */
export default function PictureControl({ spec, value, error, onChange, onWorking }: ControlProps & {
  onWorking?: (key: string, working: boolean) => void;
}) {
  const input = useRef<HTMLInputElement>(null);
  const [sending, setSending] = useState(false);
  const [refused, setRefused] = useState<Failure | null>(null);
  const src = text(value);

  async function take(file: File | undefined) {
    if (!file) return;
    setSending(true);
    setRefused(null);
    onWorking?.(spec.key, true);
    const out = await uploadFile(file);
    setSending(false);
    onWorking?.(spec.key, false);
    if (out.ok) onChange(out.value);
    else setRefused(out);
    if (input.current) input.current.value = "";
  }

  return (
    <Field label={spec.label} hint={spec.hint} group>
      <div className="flex items-center gap-4" aria-busy={sending} {...invalid(spec.key, error)}>
        <span className="flex size-24 shrink-0 items-center justify-center overflow-hidden rounded-lg border border-line bg-ground text-ink-soft">
          {src ? <img src={src} alt="" className="h-full w-full object-contain" /> : <Icon name="image" size={24} />}
        </span>
        <div className="grid gap-2">
          <div className="flex flex-wrap gap-2">
            <Button type="button" icon="photo" disabled={sending} onClick={() => input.current?.click()}>
              {src ? "Replace" : "Choose a picture"}
            </Button>
            {src && <Button type="button" kind="quiet" disabled={sending} onClick={() => onChange("")}>Remove</Button>}
          </div>
          <p className="text-xs font-normal text-ink-soft">JPEG, PNG or WebP.</p>
          <p role="status" className="text-xs font-normal text-ink-soft empty:hidden">{sending ? "Uploading…" : null}</p>
          {refused && (
            <p role="alert" id={errorId(spec.key)} className="text-xs font-normal text-danger">
              <FailureLine failure={refused} what="upload" />
            </p>
          )}
        </div>
        <input ref={input} type="file" accept="image/*" className="sr-only" tabIndex={-1}
          aria-label={`${spec.label} file`} onChange={(e) => void take(e.target.files?.[0])} />
      </div>
    </Field>
  );
}
