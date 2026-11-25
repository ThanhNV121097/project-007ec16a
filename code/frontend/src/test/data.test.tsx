// @vitest-environment jsdom
import { act, cleanup, renderHook, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { collections, setCollection, useCollection, useRecord, type Query, type Row } from "../data";

// The sandbox runs with NODE_ENV=production, whose React has no act().
vi.hoisted(() => {
  vi.stubEnv("NODE_ENV", "test");
});

// The data module against an API stubbed at fetch. Each test reads a
// collection of its own, so the module's store never carries over.

let counter = 0;
const fresh = () => `things${++counter}`;

const rows = (n: number): Row[] => Array.from({ length: n }, (_, i) => ({ id: `s${i + 1}`, name: `Shoe ${i + 1}` }));

const json = (body: unknown, status = 200) =>
  new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } });

/** The server's list answer: at most `cap` to a page, as the real one serves at most 100. */
function paged(all: Row[], url: URL, cap = 100): Response {
  const page = Number(url.searchParams.get("page") ?? 1);
  const limit = Math.min(Number(url.searchParams.get("limit") ?? 12), cap);
  return json({ items: all.slice((page - 1) * limit, page * limit), total: all.length, page, limit });
}

type Handler = (url: URL) => Response | Promise<Response>;

function serve(handler: Handler) {
  const mock = vi.fn(async (input: string) => handler(new URL(input, "http://localhost")));
  vi.stubGlobal("fetch", mock);
  return mock;
}

const asked = (mock: ReturnType<typeof serve>) => mock.mock.calls.map((c) => c[0]);

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});

describe("useCollection", () => {
  it("asks for the first page on mount, with the filter", async () => {
    const name = fresh();
    const mock = serve((url) => paged(rows(30), url));
    const { result } = renderHook(() => useCollection(name, { where: { status: "live" }, pageSize: 12 }));
    expect(result.current.loading).toBe(true);
    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(asked(mock)).toEqual([`/api/v1/${name}?page=1&limit=12&status=live`]);
    expect(result.current.items).toHaveLength(12);
    expect(result.current.total).toBe(30);
    expect(result.current.error).toBeNull();
  });

  it("appends the next page on loadMore and stops after the last", async () => {
    const name = fresh();
    const mock = serve((url) => paged(rows(30), url));
    const { result } = renderHook(() => useCollection(name, { pageSize: 12 }));
    await waitFor(() => expect(result.current.items).toHaveLength(12));
    act(() => result.current.loadMore());
    await waitFor(() => expect(result.current.items).toHaveLength(24));
    expect(result.current.hasMore).toBe(true);
    expect(asked(mock)[1]).toBe(`/api/v1/${name}?page=2&limit=12`);
    act(() => result.current.loadMore());
    await waitFor(() => expect(result.current.items).toHaveLength(30));
    expect(result.current.hasMore).toBe(false);
    act(() => result.current.loadMore());
    expect(mock).toHaveBeenCalledTimes(3);
  });

  it("turns a query into parameters", async () => {
    const name = fresh();
    const mock = serve((url) => paged([], url));
    const query: Query = {
      where: { category: ["women", "kids"], price: { from: 3000000, to: 3999999 }, size: { from: 40 }, colour: [], status: "", featured: true },
      q: "samba",
      sort: "-price",
      pageSize: 500,
    };
    const { result } = renderHook(() => useCollection(name, query));
    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(asked(mock)[0]).toBe(
      `/api/v1/${name}?page=1&limit=500&category=women,kids&price_from=3000000&price_to=3999999&size_from=40&featured=true&q=samba&sort=-price`,
    );
  });

  it("trusts the limit the server applied and its total", async () => {
    const name = fresh();
    serve((url) => paged(rows(130), url, 100));
    const { result } = renderHook(() => useCollection(name, { pageSize: 500 }));
    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.items).toHaveLength(100);
    expect(result.current.total).toBe(130);
    expect(result.current.pages).toBe(2);
    expect(result.current.hasMore).toBe(true);
  });

  it("shows one page alone when the query names a page, and the last page when it is past it", async () => {
    const name = fresh();
    const mock = serve((url) => paged(rows(30), url));
    const { result } = renderHook(() => useCollection(name, { page: 2, pageSize: 12 }));
    await waitFor(() => expect(result.current.items).toHaveLength(12));
    expect(result.current.page).toBe(2);
    expect(result.current.hasMore).toBe(false);

    const past = fresh();
    const second = serve((url) => paged(rows(30), url));
    const { result: far } = renderHook(() => useCollection(past, { page: 5, pageSize: 12 }));
    await waitFor(() => expect(far.current.items).toHaveLength(6));
    expect(far.current.page).toBe(3);
    expect(asked(second)).toEqual([`/api/v1/${past}?page=5&limit=12`, `/api/v1/${past}?page=3&limit=12`]);
    expect(mock).toHaveBeenCalledTimes(1);
  });

  it("keeps the items already loaded when a later request fails", async () => {
    const name = fresh();
    serve((url) => (url.searchParams.get("page") === "2" ? json({ error: "internal", message: "x" }, 500) : paged(rows(30), url)));
    const { result } = renderHook(() => useCollection(name, { pageSize: 12 }));
    await waitFor(() => expect(result.current.items).toHaveLength(12));
    act(() => result.current.loadMore());
    await waitFor(() => expect(result.current.error).toBeInstanceOf(Error));
    expect(result.current.loading).toBe(false);
    expect(result.current.items).toHaveLength(12);
  });

  it("reports an error and no items when the first request fails", async () => {
    const name = fresh();
    serve(() => json({ error: "internal", message: "x" }, 500));
    const { result } = renderHook(() => useCollection(name));
    await waitFor(() => expect(result.current.error).toBeInstanceOf(Error));
    expect(result.current.loading).toBe(false);
    expect(result.current.items).toEqual([]);
  });

  it("drops the old items when the query changes, and an older answer arriving last never wins", async () => {
    const name = fresh();
    let release: () => void = () => {};
    const gate = new Promise<void>((resolve) => {
      release = resolve;
    });
    serve(async (url) => {
      if (url.searchParams.get("q") === "a") {
        await gate;
        return json({ items: [{ id: "old" }], total: 1, page: 1, limit: 12 });
      }
      return json({ items: [{ id: "new" }], total: 1, page: 1, limit: 12 });
    });
    const { result, rerender } = renderHook((p: { q: string }) => useCollection(name, { q: p.q }), { initialProps: { q: "a" } });
    rerender({ q: "b" });
    await waitFor(() => expect(result.current.items.map((r) => r.id)).toEqual(["new"]));
    await act(async () => {
      release();
      await gate;
    });
    expect(result.current.items.map((r) => r.id)).toEqual(["new"]);
    expect(result.current.loading).toBe(false);
  });
});

describe("useRecord", () => {
  it("is loading until the answer, then holds the record", async () => {
    const name = fresh();
    serve(() => json({ id: "samba", name: "Samba" }));
    const { result } = renderHook(() => useRecord(name, "samba"));
    expect(result.current.loading).toBe(true);
    expect(result.current.item).toBeUndefined();
    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.item).toEqual({ id: "samba", name: "Samba" });
    expect(result.current.error).toBeNull();
  });

  it("reads a 404 as no record and no error", async () => {
    const name = fresh();
    const mock = serve(() => json({ error: "not_found", message: "Not found." }, 404));
    const { result } = renderHook(() => useRecord(name, "nope"));
    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.item).toBeUndefined();
    expect(result.current.error).toBeNull();
    expect(mock).toHaveBeenCalledTimes(1);
  });

  it("reports any other failure as an error", async () => {
    const name = fresh();
    serve(() => json({ error: "internal", message: "x" }, 500));
    const { result } = renderHook(() => useRecord(name, "samba"));
    await waitFor(() => expect(result.current.error).toBeInstanceOf(Error));
    expect(result.current.loading).toBe(false);
  });

  it("drops the old record the moment the id changes", async () => {
    const name = fresh();
    serve((url) => json({ id: url.pathname.split("/").pop(), name: "x" }));
    const { result, rerender } = renderHook((p: { id: string }) => useRecord(name, p.id), { initialProps: { id: "a" } });
    await waitFor(() => expect(result.current.item?.id).toBe("a"));
    rerender({ id: "b" });
    expect(result.current.item).toBeUndefined();
    expect(result.current.loading).toBe(true);
    await waitFor(() => expect(result.current.item?.id).toBe("b"));
  });
});

describe("the browser's own copy", () => {
  it("holds the rows loaded so far, and a set copy replaces them with no request", async () => {
    const name = fresh();
    const mock = serve((url) => paged(rows(5), url));
    const { result } = renderHook(() => useCollection(name));
    await waitFor(() => expect(result.current.items).toHaveLength(5));
    expect(collections()[name]).toHaveLength(5);
    expect(mock).toHaveBeenCalledTimes(1);

    const edited: Row[] = [{ id: "only", name: "Edited" }];
    act(() => setCollection(name, edited));
    expect(collections()[name]).toBe(edited);
    expect(result.current.items).toEqual(edited);
    expect(result.current.total).toBe(1);
    expect(mock).toHaveBeenCalledTimes(1);

    const later = renderHook(() => useCollection(name, { where: { name: "Edited" } }));
    expect(later.result.current.items).toEqual(edited);
    expect(mock).toHaveBeenCalledTimes(1);
  });
});
