import { useCallback, useEffect, useState } from "react";
import { MockFault, mockCreate, mockList, mockRemove, mockUpdate } from "./mock/staff";

/**
 * The staff page's one door to the shop (services.md §3.26, §3.27).
 *
 * Wire names are the API's: `is_active`, and `role` is "admin" or "staff".
 * At this stage the answers come from src/lib/mock/staff.ts; the backend
 * stage swaps the four calls below for src/lib/api.ts and deletes the mock,
 * and nothing that imports this file changes.
 */
export interface StaffAccount {
  id: string;
  name: string;
  email: string;
  role: "admin" | "staff";
  is_active: boolean;
}

export interface StaffInput {
  name: string;
  email: string;
  password: string;
}

export interface StaffPatch {
  name?: string;
  is_active?: boolean;
  password?: string;
}

export type StaffField = "name" | "email" | "password" | "role";

/** What a save came to: done, or refused (fields, a sentence), or never answered. */
export type StaffResult =
  | { ok: true }
  | { ok: false; kind: "fields"; errors: Partial<Record<StaffField, string>> }
  | { ok: false; kind: "refused"; message: string }
  | { ok: false; kind: "unreachable" };

export type StaffStatus = "loading" | "ready" | "error";

/** The shop's refusal or silence, as the page tells them apart. */
function failure(e: unknown): StaffResult {
  if (e instanceof MockFault && e.status === 422 && e.body.errors) {
    return { ok: false, kind: "fields", errors: e.body.errors };
  }
  if (e instanceof MockFault && e.status >= 400 && e.status < 500 && e.status !== 0) {
    return { ok: false, kind: "refused", message: e.body.message };
  }
  return { ok: false, kind: "unreachable" };
}

export function useStaff() {
  const [items, setItems] = useState<StaffAccount[]>([]);
  const [status, setStatus] = useState<StaffStatus>("loading");
  const [attempt, setAttempt] = useState(0);

  useEffect(() => {
    let live = true;
    setStatus("loading");
    mockList()
      .then((answer) => {
        if (!live) return;
        setItems(answer.items);
        setStatus("ready");
      })
      .catch(() => {
        if (live) setStatus("error");
      });
    return () => {
      live = false;
    };
  }, [attempt]);

  /** Asks again after a refusal, so the table is the shop's answer and not a guess. */
  const redraw = useCallback(() => {
    mockList().then((answer) => setItems(answer.items)).catch(() => undefined);
  }, []);

  const settle = useCallback(
    (e: unknown): StaffResult => {
      const result = failure(e);
      if (!result.ok && result.kind !== "unreachable") redraw();
      return result;
    },
    [redraw],
  );

  const add = useCallback(
    (input: StaffInput): Promise<StaffResult> =>
      mockCreate(input)
        .then((made): StaffResult => {
          setItems((cur) => [...cur, made]);
          return { ok: true };
        })
        .catch(settle),
    [settle],
  );

  const update = useCallback(
    (id: string, patch: StaffPatch): Promise<StaffResult> =>
      mockUpdate(id, patch)
        .then((saved): StaffResult => {
          setItems((cur) => cur.map((a) => (a.id === id ? saved : a)));
          return { ok: true };
        })
        .catch(settle),
    [settle],
  );

  const remove = useCallback(
    (id: string): Promise<StaffResult> =>
      mockRemove(id)
        .then((): StaffResult => {
          setItems((cur) => cur.filter((a) => a.id !== id));
          return { ok: true };
        })
        .catch(settle),
    [settle],
  );

  return { items, status, retry: () => setAttempt((n) => n + 1), add, update, remove };
}
