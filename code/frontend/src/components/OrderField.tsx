import { useId, type ReactNode } from "react";
import { cn } from "../lib/cn";
import { Field } from "./admin/index";

/** What a control must carry so the sentence beside it is tied to it. */
export interface Described {
  "aria-invalid"?: true;
  "aria-describedby"?: string;
}

/**
 * A labelled control with its hint and the shop's refusal beside it.
 *
 * The kit's Field draws a refusal as plain text that nothing points at, so a
 * screen reader never hears it and the control never says it is wrong. Here
 * the sentence is its own element with `role="alert"`, announced the moment it
 * appears, and the control it belongs to is handed `aria-invalid` and an
 * `aria-describedby` naming the hint and the sentence. The label stays the
 * kit's own, so a click on the words still reaches the control.
 */
export default function OrderField({ label, hint, error, className, children }: {
  label: string;
  hint?: string;
  error?: string;
  className?: string;
  children: (described: Described) => ReactNode;
}) {
  const id = useId();
  const hintId = `${id}-hint`;
  const errorId = `${id}-error`;
  const names = [hint ? hintId : "", error ? errorId : ""].filter((n) => n !== "").join(" ");
  return (
    <div className={cn("grid gap-2", className)}>
      <Field label={label}>
        {children({ "aria-invalid": error ? true : undefined, "aria-describedby": names === "" ? undefined : names })}
      </Field>
      {hint && <p id={hintId} className="text-xs text-ink-soft">{hint}</p>}
      {error && <p id={errorId} role="alert" className="text-xs text-danger">{error}</p>}
    </div>
  );
}
