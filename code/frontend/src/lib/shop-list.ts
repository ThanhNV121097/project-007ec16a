import { useCallback, useEffect, useState } from "react";
import { getProductFilters, listProducts } from "./search-filters-and-sorting-served-by-the-api";
import type { Product } from "./shop";
import { toRequest, type ProductFilters, type ShopQuery } from "./shop-request";

/**
 * THE SHOP'S READS: a page of live shoes that match what the visitor set, a
 * page at a time, and the colours and sizes the rail lists. This file is the
 * only door to the answers: `listProducts` and `getProductFilters` ask the API
 * and nothing in a component knows it.
 */
interface Answer {
  key: string;
  /** How many pages `items` holds. */
  pages: number;
  items: Product[];
  total: number;
}

interface Failure {
  key: string;
  pages: number;
  error: Error;
}

const NO_SHOES: Product[] = [];

/**
 * `items` are the pages loaded so far and `total` is the API's total, the
 * count the visitor reads. A new query starts again at its first page; the
 * answer to an older query that arrives late is dropped, so the newest
 * request is what shows. While a new query is on its way the last answer
 * stays on screen, and a query that failed shows no list at all.
 * `total` is null until the first answer is in: nothing is claimed empty
 * before then.
 */
export function useShopList(query: ShopQuery) {
  const key = JSON.stringify(query);
  const [paging, setPaging] = useState({ key, pages: 1 });
  const [answer, setAnswer] = useState<Answer | null>(null);
  const [failure, setFailure] = useState<Failure | null>(null);
  // A new query starts at its first page, and coming back to an old one does too.
  if (paging.key !== key) setPaging({ key, pages: 1 });
  const pages = paging.key === key ? paging.pages : 1;

  useEffect(() => {
    let current = true;
    listProducts(toRequest(JSON.parse(key) as ShopQuery, pages)).then(
      (page) => {
        if (!current) return;
        setFailure(null);
        setAnswer((prev) => {
          const before = pages > 1 && prev?.key === key ? prev.items : NO_SHOES;
          const seen = new Set(before.map((p) => p.id));
          return { key, pages, total: page.total, items: [...before, ...page.items.filter((p) => !seen.has(p.id))] };
        });
      },
      (e: unknown) => {
        if (current) setFailure({ key, pages, error: e instanceof Error ? e : new Error(String(e)) });
      },
    );
    return () => {
      current = false;
    };
  }, [key, pages]);

  const error = failure?.key === key && failure.pages === pages ? failure.error : null;
  const settled = answer?.key === key && answer.pages === pages;
  const shown = error && answer?.key !== key ? null : answer;
  const loading = !settled && !error;
  const loadMore = useCallback(() => {
    if (loading || answer?.key !== key) return;
    setPaging({ key, pages: answer.pages + 1 });
  }, [loading, answer, key]);

  return {
    items: shown?.items ?? NO_SHOES,
    total: shown ? shown.total : null,
    hasMore: shown?.key === key && shown.items.length < shown.total,
    loading,
    error,
    loadMore,
  };
}

const NO_OPTIONS: ProductFilters = { colours: [], sizes: [] };

/** The Colour options and Size (EU) pills: every one a live shoe has, however the list is filtered. */
export function useProductFilters(): ProductFilters {
  const [filters, setFilters] = useState(NO_OPTIONS);
  useEffect(() => {
    let current = true;
    getProductFilters().then(
      (f) => {
        if (current) setFilters(f);
      },
      () => {
        if (current) setFilters(NO_OPTIONS);
      },
    );
    return () => {
      current = false;
    };
  }, []);
  return filters;
}
