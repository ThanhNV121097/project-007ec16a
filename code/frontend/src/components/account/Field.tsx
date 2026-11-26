import { type InputHTMLAttributes, type ReactNode } from "react";
import Icon from "../Icon";
import { FIELD, LABEL } from "./styles";

/**
 * A sentence that says something went wrong: the danger colour, an alert icon
 * and role="alert", so it is never colour alone and a screen reader reads it
 * the moment it appears (design-system.md 2.1.16, Helper / error, 12px).
 */
export function ErrorLine({ id, children }: { id?: string; children: ReactNode }) {
  return (
    <p id={id} role="alert" className="flex items-start gap-2 text-xs text-danger">
      <Icon name="alert" size={14} className="mt-0.5" />
      <span className="min-w-0 wrap-anywhere">{children}</span>
    </p>
  );
}

/**
 * The text field of the checkout with its visible label, and its refusal
 * directly under it, linked to the field so it is read with it.
 */
export default function Field({ id, label, error, ...input }: { id: string; label: string; error?: string } & Omit<InputHTMLAttributes<HTMLInputElement>, "id">) {
  const errorId = `${id}-error`;
  return (
    <div>
      <label htmlFor={id} className={LABEL}>{label}</label>
      <input id={id} aria-invalid={error ? true : undefined} aria-describedby={error ? errorId : undefined} className={FIELD} {...input} />
      {error && <div className="mt-1.5"><ErrorLine id={errorId}>{error}</ErrorLine></div>}
    </div>
  );
}
