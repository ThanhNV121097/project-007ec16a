import { beforeEach, describe, expect, it, vi } from "vitest";

const get = vi.hoisted(() => vi.fn());
vi.mock("../lib/api", () => ({ api: { get } }));

import { getProductFilters, listProducts } from "../lib/search-filters-and-sorting-served-by-the-api";
import { toRequest } from "../lib/shop-request";

beforeEach(() => {
  get.mockReset();
});

describe("the shop's reads of the API", () => {
  it("asks GET products for twelve live shoes, with the union of bands as price_ranges", async () => {
    const answer = { items: [], total: 0, page: 1, limit: 12 };
    get.mockResolvedValue(answer);
    const request = toRequest(
      { category: ["women"], line: ["samba"], colour: [], size: ["38"], price: ["under-3m", "4m-plus"], q: "og", sort: "price" },
      1,
    );
    await expect(listProducts(request)).resolves.toBe(answer);
    expect(get).toHaveBeenCalledWith("products", {
      status: "live",
      page: "1",
      limit: "12",
      sort: "price",
      q: "og",
      category: "women",
      line: "samba",
      size: "38",
      price_ranges: "..2999999,4000000..",
    });
  });

  it("lets a failed request reject, so the list can say it failed", async () => {
    get.mockRejectedValue(new Error("The server cannot be reached."));
    await expect(listProducts(toRequest({ category: [], line: [], colour: [], size: [], price: [], q: "", sort: "-released" }, 1))).rejects.toThrow("cannot be reached");
    await expect(getProductFilters()).rejects.toThrow();
  });

  it("asks GET product-filters for the options, with no parameter", async () => {
    const options = { colours: ["black"], sizes: ["36"] };
    get.mockResolvedValue(options);
    await expect(getProductFilters()).resolves.toBe(options);
    expect(get).toHaveBeenCalledWith("product-filters");
  });
});
