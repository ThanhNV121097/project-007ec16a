// @vitest-environment jsdom
// Deleted together with the mock it covers, in plan item 4's backend step.
import { beforeEach, describe, expect, it, vi } from "vitest";

const get = vi.hoisted(() => vi.fn());
vi.mock("../lib/api", () => ({ api: { get }, ApiError: class ApiError extends Error {} }));

import { getProductFilters, listProducts } from "../lib/mock/search-filters-and-sorting-served-by-the-api";
import type { Product } from "../lib/shop";
import { DEFAULT_SORT, toRequest, type ShopQuery } from "../lib/shop-request";

const base: ShopQuery = { category: [], line: [], colour: [], size: [], price: [], q: "", sort: DEFAULT_SORT };

let tables: Record<string, object[]> = {};

function shoe(id: string, over: Partial<Product> = {}): Product {
  return { id, name: id, line: "samba", category: "women", price: 2500000, status: "live", colour: "black", sizes: ["38", "40"], ...over };
}

function many(n: number): Product[] {
  return Array.from({ length: n }, (_, i) => shoe(`s${String(i).padStart(3, "0")}`));
}

const ids = (items: Product[]) => items.map((p) => p.id);

beforeEach(() => {
  tables = {};
  get.mockReset();
  // The API as the browser meets it: a filter on a field, at most a hundred a page.
  get.mockImplementation((name: string, params: { page: number; limit: number; status?: string }) => {
    const rows = (tables[name] ?? []).filter((r) => !params.status || (r as { status?: string }).status === params.status);
    const limit = Math.min(params.limit, 100);
    return Promise.resolve({ items: rows.slice((params.page - 1) * limit, params.page * limit), total: rows.length, page: params.page, limit });
  });
});

describe("the UI stage's list, read from the API and not from what other pages loaded", () => {
  it("lists the live shoes on a direct open, a draft left out", async () => {
    tables.products = [shoe("a"), shoe("b", { status: "draft" }), shoe("c")];
    const page = await listProducts(toRequest(base, 1));
    expect(ids(page.items)).toEqual(["a", "c"]);
    expect(page).toMatchObject({ total: 2, page: 1, limit: 12 });
    expect(get).toHaveBeenCalledWith("products", expect.objectContaining({ status: "live" }));
  });

  it("pages twelve at a time and counts every match", async () => {
    tables.products = many(30);
    const first = await listProducts(toRequest(base, 1));
    expect(first.items).toHaveLength(12);
    expect(first.total).toBe(30);
    expect((await listProducts(toRequest(base, 3))).items).toHaveLength(6);
  });

  it("reads past the API's hundred-a-page ceiling", async () => {
    tables.products = many(130);
    const last = await listProducts(toRequest(base, 11));
    expect(last.total).toBe(130);
    expect(ids(last.items)).toEqual(ids(many(130).slice(120)));
  });

  it("narrows by who it is for, the price and an EU size still in stock", async () => {
    tables.products = [
      shoe("a"),
      shoe("b", { category: "men", price: 3500000, sizes: ["42"] }),
      shoe("c", { sizes_out: ["38", "40"] }),
    ];
    expect(ids((await listProducts(toRequest({ ...base, size: ["38"] }, 1))).items)).toEqual(["a"]);
    expect(ids((await listProducts(toRequest({ ...base, category: ["men"] }, 1))).items)).toEqual(["b"]);
    expect(ids((await listProducts(toRequest({ ...base, price: ["3m-4m"] }, 1))).items)).toEqual(["b"]);
  });

  it("finds a word in a line's name, asking for the names only when a word was typed", async () => {
    tables.products = [shoe("a", { name: "Ultra" }), shoe("b", { name: "Boost", line: "gazelle" })];
    tables.lines = [{ id: "samba", name: "Samba" }, { id: "gazelle", name: "Gazelle" }];
    tables.categories = [{ id: "women", name: "Women" }];
    await listProducts(toRequest(base, 1));
    expect(get).not.toHaveBeenCalledWith("lines", expect.anything());
    expect(ids((await listProducts(toRequest({ ...base, q: "samba" }, 1))).items)).toEqual(["a"]);
  });

  it("rejects when the API cannot be read, never an empty list", async () => {
    get.mockRejectedValue(new Error("500"));
    await expect(listProducts(toRequest(base, 1))).rejects.toThrow("500");
  });
});

describe("the UI stage's options", () => {
  it("lists the colours and sizes of the live shoes on a direct open", async () => {
    tables.products = [
      shoe("a", { sizes: ["40", "38"] }),
      shoe("b", { colour: "white", sizes: ["36 2/3"] }),
      shoe("c", { colour: "pink", status: "draft" }),
    ];
    expect(await getProductFilters()).toEqual({ colours: ["black", "white"], sizes: ["36 2/3", "38", "40"] });
  });
});
