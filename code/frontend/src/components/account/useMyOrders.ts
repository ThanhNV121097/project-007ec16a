import { useCallback, useEffect, useRef, useState } from "react";
import { listOrders, refreshMe, type OrderRow } from "../../lib/customer-accounts-sign-up-sign-in-my-orders";

export type Phase = "loading" | "ready" | "failed";

export interface MyOrders {
  phase: Phase;
  items: OrderRow[];
  hasMore: boolean;
  loadingMore: boolean;
  /** The next ten could not be fetched; the ones already shown stay. */
  moreFailed: boolean;
  showMore: () => void;
  /** Asks for the first ten again, with "Loading your orders…" while it waits. */
  retry: () => void;
  /** Asks for the first ten again without a loading view: a new order was added. */
  reload: () => void;
}

/**
 * The customer's own orders, newest first, ten at a time. A 401 or 403 means
 * the session ended, so it asks /v1/auth/me again and the page takes the
 * visitor to sign-in; any other failure is "could not load" with a way to try
 * again.
 */
export function useMyOrders(): MyOrders {
  const [phase, setPhase] = useState<Phase>("loading");
  const [items, setItems] = useState<OrderRow[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [loadingMore, setLoadingMore] = useState(false);
  const [moreFailed, setMoreFailed] = useState(false);
  const live = useRef(true);

  const first = useCallback(async (quiet: boolean) => {
    if (!quiet) setPhase("loading");
    const out = await listOrders(1);
    if (!live.current) return;
    if (out.kind === "ok") {
      setItems(out.value.items);
      setTotal(out.value.total);
      setPage(1);
      setMoreFailed(false);
      setPhase("ready");
      return;
    }
    if (out.kind === "unauthenticated" || out.kind === "forbidden") void refreshMe();
    if (!quiet) setPhase("failed");
  }, []);

  const more = useCallback(async () => {
    setLoadingMore(true);
    setMoreFailed(false);
    const out = await listOrders(page + 1);
    if (!live.current) return;
    setLoadingMore(false);
    if (out.kind !== "ok") {
      setMoreFailed(true);
      return;
    }
    setItems((shown) => [...shown, ...out.value.items]);
    setTotal(out.value.total);
    setPage(page + 1);
  }, [page]);

  useEffect(() => {
    live.current = true;
    void first(false);
    return () => {
      live.current = false;
    };
  }, [first]);

  return {
    phase,
    items,
    hasMore: items.length < total,
    loadingMore,
    moreFailed,
    showMore: () => void more(),
    retry: () => void first(false),
    reload: () => void first(true),
  };
}
