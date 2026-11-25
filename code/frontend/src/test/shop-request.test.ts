import { describe, expect, it } from "vitest";
import { bandRanges, DEFAULT_SORT, PAGE_SIZE, readShopQuery, toRequest, type ShopQuery } from "../lib/shop-request";

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
