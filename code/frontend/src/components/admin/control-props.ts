import type { FieldSpec } from "./fields";

/** What every control of the form is given. */
export type ControlProps = {
  spec: FieldSpec;
  value: unknown;
  /** The shop's sentence for this field. The form draws it under the control. */
  error?: string;
  onChange: (value: unknown) => void;
};

/** Where a field's sentence is drawn, so the control can point at it. */
export const errorId = (key: string) => `field-${key}-error`;

/** The attributes that mark a control as the one that failed. */
export function invalid(key: string, error?: string) {
  return error ? { "aria-invalid": true as const, "aria-describedby": errorId(key) } : {};
}
