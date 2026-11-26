import { useCallback, useRef, useState } from "react";
import type { Failure, Outcome } from "./store";

/**
 * THE STATE OF ONE SAVE OR DELETE, shared by the dialog, the record page and
 * the delete question so they cannot disagree about it.
 *
 * `run` takes a job that answers with an outcome and does three things the
 * screens would otherwise each get slightly wrong: it sends nothing while a
 * job is already out (a second press, Enter in a field), it drops the last
 * refusal when a new attempt starts, and it keeps the shop's field errors and
 * its other refusals apart — the first sit beside their fields, the second in
 * a note.
 */
export function useAction() {
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState<Failure | null>(null);
  const [errors, setErrors] = useState<Record<string, string>>({});
  const out = useRef(false);

  const run = useCallback(async <T,>(job: () => Promise<Outcome<T>>): Promise<Outcome<T> | null> => {
    if (out.current) return null;
    out.current = true;
    setBusy(true);
    setFailure(null);
    setErrors({});
    try {
      const result = await job();
      if (!result.ok) {
        setFailure(result);
        if (result.kind === "fields") setErrors(result.errors);
      }
      return result;
    } finally {
      out.current = false;
      setBusy(false);
    }
  }, []);

  /**
   * Refusals found before anything was sent. They are added to what the shop
   * has already said about fields that have not changed since: a sentence
   * leaves only when its own field is edited (`clear`), so a blank name found
   * here does not wipe "Price cannot be below 0" from a price left as it was.
   */
  const refuse = useCallback((found: Record<string, string>) => {
    setFailure(null);
    setErrors((cur) => ({ ...cur, ...found }));
  }, []);

  /** The person changed a field: its refusal (and its sizes' counts', for `pairs`) is out of date. */
  const clear = useCallback((key: string) => {
    setErrors((cur) => {
      const next = Object.fromEntries(Object.entries(cur).filter(([k]) => k !== key && !k.startsWith(`${key}.`)));
      return Object.keys(next).length === Object.keys(cur).length ? cur : next;
    });
  }, []);

  return { busy, failure, errors, run, refuse, clear };
}

/**
 * Which controls of a form are still sending a file. Save waits for them: a
 * record saved before its picture's address is back would lose the picture.
 */
export function useWorking() {
  const [on, setOn] = useState<ReadonlySet<string>>(new Set());
  const change = useCallback((key: string, working: boolean) => {
    setOn((cur) => {
      const next = new Set(cur);
      if (working) next.add(key);
      else next.delete(key);
      return next;
    });
  }, []);
  return { working: on.size > 0, change };
}
