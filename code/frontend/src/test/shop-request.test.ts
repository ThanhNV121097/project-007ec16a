import { describe, expect, it } from "vitest";
<<<<<<< refs/heads/feature/shop-phone-filters-clear-all-loading-and-errors-ui
import { bandRanges, DEFAULT_SORT, PAGE_SIZE, readShopQuery, toRequest, type ShopQuery } from "../lib/shop-request";
=======
import { bandRanges, DEFAULT_SORT, PAGE_SIZE, readShopQuery, toParams, toRequest, type ShopQuery } from "../lib/shop-request";
>>>>>>> refs/heads/main

/** What the address bar holds, as useTableQuery reads it. */
function address(search: string, q = "", sort = "") {
  const params = new URLSearchParams(search);
  return {
    q,
    sort,
    list: (name: string) => (params.get(name) ?? "").split(",").map((v) => v.trim()).filter(Boolean),
  };
}

describe("the shop's request", () => {
  it("reads every filter in the address, and size as an EU size", () => {
    const query = readShopQuery(address("category=women,men&line=samba&colour=black&size=38,40&price=3m-4m"));
    expect(query).toEqual({
      category: ["women", "men"],
      line: ["samba"],
      colour: ["black"],
      size: ["38", "40"],
      price: ["3m-4m"],
      q: "",
      sort: DEFAULT_SORT,
    });
  });

  it("treats an unknown price band and an unknown sort as if they were not set", () => {
    const query = readShopQuery(address("price=abc", "", "zzz"));
    expect(query.price).toEqual([]);
    expect(query.sort).toBe(DEFAULT_SORT);
  });

  it("keeps a sort the shop offers and trims the words searched", () => {
    const query = readShopQuery(address("", "  samba og ", "-price"));
    expect(query.sort).toBe("-price");
    expect(query.q).toBe("samba og");
  });

  it("carries ticked price bands as a union of ranges", () => {
    expect(bandRanges(["under-3m", "4m-plus"])).toEqual([{ to: 2999999 }, { from: 4000000 }]);
    expect(bandRanges(["3m-4m"])).toEqual([{ from: 3000000, to: 3999999 }]);
    expect(bandRanges([])).toEqual([]);
  });

  it("always asks for twelve live shoes, whatever size the address holds", () => {
    const query: ShopQuery = readShopQuery(address("size=38"));
    const request = toRequest(query, 2);
    expect(request.limit).toBe(PAGE_SIZE);
    expect(request.limit).toBe(12);
    expect(request.status).toBe("live");
    expect(request.page).toBe(2);
    expect(request.size).toEqual(["38"]);
  });
});
<<<<<<< refs/heads/feature/shop-phone-filters-clear-all-loading-and-errors-ui
=======

describe("the query string the API is asked", () => {
  it("asks for the first twelve live shoes in the default order when nothing is set", () => {
    expect(toParams(toRequest(readShopQuery(address("")), 1))).toEqual({
      status: "live",
      page: "1",
      limit: "12",
      sort: "-released",
    });
  });

  it("writes each list as a comma list and the words, the order and the page as they are", () => {
    const query = readShopQuery(address("category=women,men&line=samba&colour=black&size=36 2/3,38", "samba og", "-price"));
    expect(toParams(toRequest(query, 2))).toEqual({
      status: "live",
      page: "2",
      limit: "12",
      sort: "-price",
      q: "samba og",
      category: "women,men",
      line: "samba",
      colour: "black",
      size: "36 2/3,38",
    });
  });

  it("writes a union of price bands as one price_ranges, and a single band as one range", () => {
    const both = toParams(toRequest(readShopQuery(address("price=under-3m,4m-plus")), 1));
    expect(both.price_ranges).toBe("..2999999,4000000..");
    const middle = toParams(toRequest(readShopQuery(address("price=3m-4m")), 1));
    expect(middle.price_ranges).toBe("3000000..3999999");
    expect(middle).not.toHaveProperty("price_from");
    expect(middle).not.toHaveProperty("price_to");
  });

  it("keeps limit at twelve whatever size holds: size is the EU size, never the page size", () => {
    const params = toParams(toRequest(readShopQuery(address("size=38")), 3));
    expect(params.limit).toBe("12");
    expect(params.size).toBe("38");
    expect(params.page).toBe("3");
  });

  it("leaves an unknown price band and an unknown sort out of the request", () => {
    const params = toParams(toRequest(readShopQuery(address("price=abc", "", "zzz")), 1));
    expect(params).not.toHaveProperty("price_ranges");
    expect(params.sort).toBe("-released");
  });
});
>>>>>>> refs/heads/main
