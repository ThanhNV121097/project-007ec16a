// @vitest-environment jsdom
import { act, renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { Product } from "../lib/shop";
import { DEFAULT_SORT, type Page, type ProductsRequest, type ShopQuery } from "../lib/shop-request";

// React's act() needs the development build; the sandbox may export production.
const calls = vi.hoisted(() => {
  vi.stubEnv("NODE_ENV", "test");
  return [] as { req: ProductsRequest; resolve: (p: Page<Product>) => void; reject: (e: Error) => void }[];
});

vi.mock("../lib/search-filters-and-sorting-served-by-the-api", () => ({
  listProducts: (req: ProductsRequest) => new Promise<Page<Product>>((resolve, reject) => {
    calls.push({ req, resolve, reject });
  }),
  getProductFilters: () => Promise.resolve({ colours: [], sizes: [] }),
}));

import { useShopList } from "../lib/shop-list";

const base: ShopQuery = { category: [], line: [], colour: [], size: [], price: [], q: "", sort: DEFAULT_SORT };

function shoe(id: string): Product {
  return { id, name: id, line: "samba", category: "women", price: 1 };
}

function page(ids: string[], total: number, n = 1): Page<Product> {
  return { items: ids.map(shoe), total, page: n, limit: 12 };
}

function setup(query: ShopQuery = base) {
  calls.length = 0;
  return renderHook(({ q }) => useShopList(q), { initialProps: { q: query } });
}

describe("the shop's list", () => {
  it("claims nothing until the first answer, then reports the API's total", async () => {
    const { result } = setup();
    expect(calls[0].req).toMatchObject({ status: "live", limit: 12, page: 1 });
    expect(result.current).toMatchObject({ items: [], total: null, loading: true, hasMore: false });
    await act(async () => calls[0].resolve(page(["a", "b"], 30)));
    expect(result.current.items.map((p) => p.id)).toEqual(["a", "b"]);
    expect(result.current).toMatchObject({ total: 30, loading: false, hasMore: true });
  });

  it("adds the next page below, without a shoe twice, until every match is shown", async () => {
    const { result } = setup();
    await act(async () => calls[0].resolve(page(["a", "b"], 3)));
    act(() => result.current.loadMore());
    expect(calls[1].req.page).toBe(2);
    await act(async () => calls[1].resolve(page(["b", "c"], 3, 2)));
    expect(result.current.items.map((p) => p.id)).toEqual(["a", "b", "c"]);
    expect(result.current.hasMore).toBe(false);
  });

  it("shows the newest request's answer when an older one arrives late", async () => {
    const { result, rerender } = setup();
    rerender({ q: { ...base, size: ["38"] } });
    expect(calls[1].req.size).toEqual(["38"]);
    await act(async () => calls[1].resolve(page(["new"], 1)));
    await act(async () => calls[0].resolve(page(["old", "older"], 2)));
    expect(result.current.items.map((p) => p.id)).toEqual(["new"]);
    expect(result.current.total).toBe(1);
  });

  it("returns to the first twelve when a filter changes", async () => {
    const { result, rerender } = setup();
    await act(async () => calls[0].resolve(page(["a"], 30)));
    act(() => result.current.loadMore());
    await act(async () => calls[1].resolve(page(["b"], 30, 2)));
    rerender({ q: { ...base, category: ["women"] } });
    expect(calls[calls.length - 1].req).toMatchObject({ page: 1, category: ["women"] });
    await act(async () => calls[calls.length - 1].resolve(page(["w"], 1)));
    expect(result.current.items.map((p) => p.id)).toEqual(["w"]);
  });

  it("starts an earlier query again from its first page", async () => {
    const { result, rerender } = setup();
    await act(async () => calls[0].resolve(page(["a"], 30)));
    act(() => result.current.loadMore());
    await act(async () => calls[1].resolve(page(["b"], 30, 2)));
    rerender({ q: { ...base, category: ["men"] } });
    await act(async () => calls[calls.length - 1].resolve(page(["m"], 1)));
    rerender({ q: base });
    expect(calls[calls.length - 1].req).toMatchObject({ page: 1, category: [] });
    await act(async () => calls[calls.length - 1].resolve(page(["a"], 30)));
    expect(result.current.items.map((p) => p.id)).toEqual(["a"]);
  });

  it("shows no list and no total for a request that failed", async () => {
    const { result } = setup();
    await act(async () => calls[0].reject(new Error("500")));
    expect(result.current).toMatchObject({ items: [], total: null, loading: false });
    expect(result.current.error?.message).toBe("500");
  });
});
