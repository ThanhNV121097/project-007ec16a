import { useRef, useState, type ReactNode } from "react";
import type { Row } from "../../data";
import { cn } from "../../lib/cn";
import { errorId, invalid } from "./control-props";
import { Field, Input } from "./index";
import { WHOLE_COUNT, slugOf } from "./payload";
import { FailureLine } from "./SaveNote";
import { saveCount, type SizeAnswer } from "./store";

/**
 * The shoe's page lets each existing size's count save by itself. This is
 * what it needs from the page: which shoe, the counts as the shop last said
 * them, and what to do with the shop's answer.
 */
export interface StockLink {
  productId: string;
  known: Record<string, string>;
  onAnswer(answer: SizeAnswer): void;
  /** An order moved the count: the shoe as the shop has it now, when it could be read. */
  onStale(size: string, record: Row | null): void;
}

type Note = { danger: boolean; text: ReactNode } | null;

/**
 * ONE SIZE'S COUNT: a labelled whole number. On a saved shoe's page it saves
 * by itself when the person leaves the field or presses Enter — and sends
 * nothing when the number is the one the shop has — saying "Saving…", then
 * "Saved.", or why not. The count sent carries the one the person last saw,
 * so the shop can refuse it when an order has moved it since.
 */
function CountRow({ size, value, error, onType, link }: {
  size: string;
  value: string;
  error?: string;
  onType: (text: string) => void;
  link?: StockLink;
}) {
  const [note, setNote] = useState<Note>(null);
  const out = useRef(false);
  const key = `pairs-${slugOf(size)}`;

  async function commit() {
    const seen = link?.known[size];
    const typed = value.trim();
    if (!link || out.current || seen === undefined || typed === seen) return;
    if (!/^\d+$/.test(typed)) {
      setNote({ danger: true, text: WHOLE_COUNT });
      return;
    }
    out.current = true;
    setNote({ danger: false, text: "Saving…" });
    const result = await saveCount(link.productId, { size, pairs: Number(typed), seen_pairs: Number(seen) });
    out.current = false;
    if (result.ok) {
      link.onAnswer(result.value);
      setNote({ danger: false, text: "Saved." });
      return;
    }
    if (result.kind === "stale") link.onStale(size, result.record);
    setNote({ danger: true, text: <FailureLine failure={result} /> });
  }

  return (
    <div className="grid gap-1.5">
      <Field label={`Pairs in EU ${size}`}>
        <Input type="number" inputMode="numeric" min={0} step={1} value={value}
          className="h-10 tabular-nums" {...invalid(key, error)}
          onChange={(e) => { setNote(null); onType(e.target.value); }}
          onBlur={() => void commit()}
          onKeyDown={(e) => { if (e.key === "Enter") { e.preventDefault(); void commit(); } }} />
      </Field>
      {error && <p id={errorId(key)} role="alert" className="text-xs text-danger">{error}</p>}
      <p role="status" className={cn("text-xs empty:hidden", note?.danger ? "text-danger" : "text-ink-soft")}>
        {note?.text}
      </p>
    </div>
  );
}

/** A count for each size the shoe has, one column on a phone and two from `sm`. */
export default function StockCounts({ sizes, counts, errors, onCount, link }: {
  sizes: readonly string[];
  counts: Record<string, string>;
  errors: Record<string, string>;
  onCount: (size: string, text: string) => void;
  link?: StockLink;
}) {
  if (sizes.length === 0) return null;
  return (
    <div className="grid gap-4 sm:grid-cols-2">
      {sizes.map((size) => (
        <CountRow key={size} size={size} value={counts[size] ?? ""} error={errors[`pairs.${size}`]}
          onType={(text) => onCount(size, text)} link={link} />
      ))}
    </div>
  );
}
