import { useCallback, useEffect, useState } from "react";
import { ApiError, api } from "./api";

/**
 * The staff page's one door to the shop (services §3.26, §3.27), over the
 * project's API client.
 *
 * Wire names are the API's: `is_active`, and `role` is "admin" or "staff".
 * The page asks for up to 100 accounts and draws what it gets; the answer's
 * `total` is kept so the table can say how many there are in all.
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

interface StaffList {
  items: StaffAccount[];
  total: number;
}

const LIMIT = 100;

/**
 * The shop's refusal or silence, as the page tells them apart: a 422 names
 * fields; any other 4xx is a sentence for the open dialog; no answer, a timeout
 * and a 5xx are "unreachable" and the dialog says it could not save.
 */
function failure(e: unknown): StaffResult {
  if (e instanceof ApiError) {
    if (e.status === 422 && e.errors) return { ok: false, kind: "fields", errors: e.errors };
    if (e.status >= 400 && e.status < 500) return { ok: false, kind: "refused", message: e.message };
  }
  return { ok: false, kind: "unreachable" };
}

export function useStaff() {
  const [items, setItems] = useState<StaffAccount[]>([]);
  const [total, setTotal] = useState(0);
  const [status, setStatus] = useState<StaffStatus>("loading");
  const [attempt, setAttempt] = useState(0);

  useEffect(() => {
    let live = true;
    setStatus("loading");
    api
      .get<StaffList>("admin/staff", { limit: LIMIT })
      .then((answer) => {
        if (!live) return;
        setItems(answer.items);
        setTotal(answer.total);
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
    api
      .get<StaffList>("admin/staff", { limit: LIMIT })
      .then((answer) => {
        setItems(answer.items);
        setTotal(answer.total);
      })
      .catch(() => undefined);
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
      api
        .post<StaffAccount>("admin/staff", input)
        .then((made): StaffResult => {
          setItems((cur) => [...cur, made]);
          setTotal((n) => n + 1);
          return { ok: true };
        })
        .catch(settle),
    [settle],
  );

  const update = useCallback(
    (id: string, patch: StaffPatch): Promise<StaffResult> =>
      api
        .put<StaffAccount>(`admin/staff/${id}`, patch)
        .then((saved): StaffResult => {
          setItems((cur) => cur.map((a) => (a.id === id ? saved : a)));
          return { ok: true };
        })
        .catch(settle),
    [settle],
  );

  const remove = useCallback(
    (id: string): Promise<StaffResult> =>
      api
        .delete(`admin/staff/${id}`)
        .then((): StaffResult => {
          setItems((cur) => cur.filter((a) => a.id !== id));
          setTotal((n) => Math.max(0, n - 1));
          return { ok: true };
        })
        .catch(settle),
    [settle],
  );

  return { items, total, status, retry: () => setAttempt((n) => n + 1), add, update, remove };
}
