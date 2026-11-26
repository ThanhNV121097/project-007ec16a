// @vitest-environment jsdom
import { renderHook, act } from "@testing-library/react";
import { type ReactNode } from "react";
import { MemoryRouter, useLocation } from "react-router-dom";
import { describe, expect, it, vi } from "vitest";

// React's act() needs the development build; the sandbox may export production.
vi.hoisted(() => {
  vi.stubEnv("NODE_ENV", "test");
});

import { useTableQuery } from "../lib/table";

type Options = Parameters<typeof useTableQuery>[0];

function setup(address: string, options?: Options) {
  const wrapper = ({ children }: { children: ReactNode }) => <MemoryRouter initialEntries={[address]}>{children}</MemoryRouter>;
  return renderHook(() => ({ t: useTableQuery(options), search: useLocation().search }), { wrapper });
}

/** The parameters in the address, as a set: their order is not part of the answer. */
function params(search: string): string[] {
  return [...new URLSearchParams(search)].map(([k, v]) => `${k}=${v}`).sort();
}

describe("a list that does not opt in (every dealer list)", () => {
  it("reads size as its page size", () => {
    const { result } = setup("/admin/products?size=50&status=live", { size: 20 });
    expect(result.current.t.size).toBe(50);
  });

  it("does not count size as a filter", () => {
    expect(setup("/x?size=50").result.current.t.filtering).toBe(false);
    expect(setup("/x?size=50&page=2&sort=price").result.current.t.filtering).toBe(false);
    expect(setup("/x?size=50&status=live").result.current.t.filtering).toBe(true);
  });

  it("clears every filter and the search, and keeps size and the order", () => {
    const { result } = setup("/x?size=50&status=live&q=a&page=2&sort=price");
    act(() => result.current.t.clear());
    expect(params(result.current.search)).toEqual(["size=50", "sort=price"]);
  });
});

describe("the shop's list (size is an EU size)", () => {
  const shop = { sizeIsFilter: true };

  it("never reads size as a page size", () => {
    expect(setup("/shop?size=38", shop).result.current.t.size).toBe(20);
    expect(setup("/shop?size=38", { ...shop, size: 12 }).result.current.t.size).toBe(12);
  });

  it("counts size as a filter, alone or with others", () => {
    expect(setup("/shop?size=38", shop).result.current.t.filtering).toBe(true);
    expect(setup("/shop", shop).result.current.t.filtering).toBe(false);
    expect(setup("/shop?sort=price", shop).result.current.t.filtering).toBe(false);
    expect(setup("/shop?q=samba", shop).result.current.t.filtering).toBe(true);
  });

  it("clears size with every other filter and the search, and keeps only the order", () => {
    const { result } = setup("/shop?category=women&line=samba&size=38&price=3m-4m&q=og&sort=price", shop);
    act(() => result.current.t.clear());
    expect(params(result.current.search)).toEqual(["sort=price"]);
  });

  it("clears a lone size", () => {
    const { result } = setup("/shop?size=38", shop);
    act(() => result.current.t.clear());
    expect(result.current.search).toBe("");
  });
});
