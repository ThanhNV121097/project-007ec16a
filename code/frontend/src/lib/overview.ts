import { useCallback, useEffect, useState } from "react";
import { readOverview } from "./mock/dealer-dashboard-figures-and-charts";

/**
 * The overview's one door to the shop (services §3.25): every figure and chart
 * of /admin in one answer, added up by the shop. Wire names are the API's.
 *
 * Money and counts are whole numbers; `date` is the shop's own day as text
 * ("2026-11-12"), cut at Vietnam midnight; `placed_at` is RFC 3339 with +07:00.
 */
export interface LowShoe {
  id: string;
  name: string;
  /** The category's id. */
  category: string;
  image: string;
  /** Pairs left in all. */
  stock: number;
  /** The sizes down to one or two pairs. */
  sizes_low: string[];
}

export interface DaySales {
  date: string;
  sales: number;
}

export interface LinePairs {
  line: string;
  name: string;
  pairs: number;
}

/** One of the five newest orders: no phone, email, address or note. */
export interface LatestOrder {
  id: string;
  number: string;
  customer: string;
  product: string;
  product_name: string;
  /** The size of the order's first pair. */
  size: string;
  status: string;
  placed_at: string;
  total: number;
}

export interface Overview {
  month: { sales: number; orders: number };
  waiting_orders: number;
  pairs_on_shelf: number;
  shoes: number;
  running_low: LowShoe[];
  sales_by_day: DaySales[];
  pairs_by_line: LinePairs[];
  latest_orders: LatestOrder[];
}

export type OverviewState =
  | { status: "loading" }
  | { status: "error" }
  | { status: "ready"; data: Overview };

/** The one read. At the backend step this becomes api.get<unknown>("admin/overview"). */
const read: () => Promise<unknown> = readOverview;

function hasFigures(v: Partial<Overview>): boolean {
  return (
    typeof v.month?.sales === "number" &&
    typeof v.month.orders === "number" &&
    typeof v.waiting_orders === "number" &&
    typeof v.pairs_on_shelf === "number" &&
    typeof v.shoes === "number"
  );
}

function hasLists(v: Partial<Overview>): boolean {
  return (
    Array.isArray(v.running_low) &&
    Array.isArray(v.sales_by_day) &&
    Array.isArray(v.pairs_by_line) &&
    Array.isArray(v.latest_orders)
  );
}

/** An answer that is not the overview's shape is a failed read, never a page of zeros. */
function isOverview(answer: unknown): answer is Overview {
  if (typeof answer !== "object" || answer === null) return false;
  const v = answer as Partial<Overview>;
  return hasFigures(v) && hasLists(v);
}

/**
 * The day of the month of the shop's own date text: "2026-11-12" → "12",
 * "2026-12-01" → "1". Read from the text, never through a Date, so a dealer
 * whose laptop is set to another zone sees the days the shop counted.
 */
export function dayOfMonth(date: string): string {
  return String(Number(date.slice(8, 10)));
}

/**
 * The overview as the page needs it: loading, ready with the answer, or error
 * for anything that is not an answer — a refusal, a failure, no connection,
 * a body of the wrong shape. Every visit asks again; "retry" asks once more
 * and shows loading while it waits.
 */
export function useOverview(): OverviewState & { retry: () => void } {
  const [attempt, setAttempt] = useState(0);
  const [done, setDone] = useState<{ attempt: number; data: Overview | null } | null>(null);

  useEffect(() => {
    let live = true;
    read()
      .then((answer) => {
        if (live) setDone({ attempt, data: isOverview(answer) ? answer : null });
      })
      .catch(() => {
        if (live) setDone({ attempt, data: null });
      });
    return () => {
      live = false;
    };
  }, [attempt]);

  const retry = useCallback(() => setAttempt((n) => n + 1), []);

  if (done === null || done.attempt !== attempt) return { status: "loading", retry };
  if (done.data === null) return { status: "error", retry };
  return { status: "ready", data: done.data, retry };
}
