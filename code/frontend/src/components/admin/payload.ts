/**
 * WHAT A FORM SENDS, AND WHAT IT CHECKS FIRST.
 *
 * A form holds typed text and the shop wants a record's own fields, so this
 * file is the one place that turns one into the other:
 *
 *   - a new record sends what was filled in, and always its status;
 *   - a change sends ONLY the fields the person changed, so a save of the
 *     price alone cannot touch anything else, and a field the shop derives
 *     (a rating, the lists of sizes, the stock total) is never sent at all;
 *   - the counts of the sizes go as `pairs` only when a count or a size was
 *     touched, together with `seen_pairs`, the counts the form last read, so
 *     the shop can refuse a count an order has moved since. A form in which
 *     no count was touched carries neither.
 *
 * `seen_pairs` is this client's name for that guard; the technical design may
 * rename it, and this file is the one place it is written.
 */
import type { Row } from "../../data";
import type { FieldSpec } from "./fields";

export type Values = Record<string, unknown>;

export const EMPTY_COUNT = "Enter the pairs in this size";
export const WHOLE_COUNT = "Use a whole number, 0 or more";

/** A list as the form holds it: tags, or text with commas. */
export function listOf(value: unknown): string[] {
  if (Array.isArray(value)) return value.map(String);
  if (typeof value === "string") return value.split(",").map((s) => s.trim()).filter(Boolean);
  return [];
}

/** A size as people write it (`36 2/3`, `35.5`): spaces collapsed. The shop judges the rest. */
export function cleanSize(size: string): string {
  return size.replace(/\s+/g, " ").trim();
}

/** A size in an id: `36 2/3` → `36-2-3`. */
export function slugOf(size: string): string {
  return size.replace(/[^a-z0-9]+/gi, "-");
}

/** The counts the shop holds for a shoe, as the text a field shows. */
export function countsOf(row: Row | undefined): Record<string, string> {
  const held = row?.pairs;
  if (!held || typeof held !== "object") return {};
  return Object.fromEntries(Object.entries(held as Record<string, unknown>).map(([size, n]) => [size, String(n)]));
}

/** A record as the form holds it. */
export function valuesOf(row: Row): Values {
  return { ...row, pairs: countsOf(row) };
}

/** Counts that are empty or not whole numbers, by `pairs.<size>`. A count never read is not blamed. */
export function stockErrors(values: Values, record?: Row): Record<string, string> {
  const out: Record<string, string> = {};
  const typed = (values.pairs ?? {}) as Record<string, unknown>;
  const known = countsOf(record);
  const before = listOf(record?.sizes);
  for (const size of listOf(values.sizes)) {
    const text = String(typed[size] ?? "").trim();
    if (text === "" && (!before.includes(size) || size in known)) out[`pairs.${size}`] = EMPTY_COUNT;
    else if (text !== "" && !/^\d+$/.test(text)) out[`pairs.${size}`] = WHOLE_COUNT;
  }
  return out;
}

function stockBody(values: Values, record?: Row): Record<string, unknown> {
  const sizes = listOf(values.sizes);
  const before = listOf(record?.sizes);
  const typed = (values.pairs ?? {}) as Record<string, unknown>;
  const known = countsOf(record);
  const same = sizes.length === before.length && sizes.every((s) => before.includes(s));
  const touched = !same || sizes.some((s) => String(typed[s] ?? "").trim() !== (known[s] ?? ""));
  if (!touched || (!record && sizes.length === 0)) return {};
  const pairs = Object.fromEntries(sizes.map((s) => [s, Number(String(typed[s]).trim())]));
  if (!record) return { pairs };
  return { pairs, seen_pairs: Object.fromEntries(Object.entries(known).map(([s, n]) => [s, Number(n)])) };
}

function text(f: FieldSpec, v: unknown): string {
  if (v === undefined || v === null) return f.type === "bool" ? "false" : "";
  return Array.isArray(v) ? v.map(String).join("\u0000") : String(v);
}

const numeric = (f: FieldSpec) => f.type === "number" || f.type === "price";

function send(f: FieldSpec, v: unknown): unknown {
  if (f.type === "bool") return Boolean(v);
  if (Array.isArray(v)) return v.map(String);
  if (v === undefined || v === null || v === "") return numeric(f) ? null : "";
  return numeric(f) ? Number(v) : v;
}

const filled = (v: unknown) => !(v === undefined || v === null || v === "" || v === false || (Array.isArray(v) && v.length === 0));

/** The body of a POST (no record) or of a PUT (the record as the shop last said it). */
export function bodyOf(specs: readonly FieldSpec[], values: Values, record?: Row): Record<string, unknown> {
  const body: Record<string, unknown> = {};
  for (const f of specs) {
    if (f.readOnly || f.type === "sizes") continue;
    const now = values[f.key];
    if (record ? text(f, now) !== text(f, record[f.key]) : filled(now)) body[f.key] = send(f, now);
  }
  return { ...body, ...stockBody(values, record) };
}
