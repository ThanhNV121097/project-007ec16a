import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { api } from "../lib/api";

// The one optional trailing `headers` parameter of api.get and api.post: an order's
// token travels in a header and nowhere else, and two reads share a request only
// when their address AND their headers are the same.

const fetchMock = vi.fn();

beforeEach(() => {
  fetchMock.mockReset();
  fetchMock.mockImplementation(() =>
    Promise.resolve(new Response(JSON.stringify({ ok: true }), { status: 200, headers: { "Content-Type": "application/json" } })),
  );
  vi.stubGlobal("fetch", fetchMock);
});

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("extra request headers", () => {
  it("adds them to a GET beside Accept and keeps them out of the address", async () => {
    await api.get("orders/doi-1", undefined, { "X-Order-Token": "secret-token" });
    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(url).toBe("/api/v1/orders/doi-1");
    expect(url).not.toContain("secret-token");
    expect(init.headers).toEqual({ Accept: "application/json", "X-Order-Token": "secret-token" });
  });

  it("sends only Accept when a call names none", async () => {
    await api.get("orders/doi-2");
    const [, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(init.headers).toEqual({ Accept: "application/json" });
  });

  it("makes two requests for the same address with different headers, and one for the same headers", async () => {
    await Promise.all([
      api.get("orders/doi-3", undefined, { "X-Order-Token": "one" }),
      api.get("orders/doi-3", undefined, { "X-Order-Token": "two" }),
    ]);
    expect(fetchMock).toHaveBeenCalledTimes(2);
    fetchMock.mockClear();
    await Promise.all([
      api.get("orders/doi-4", undefined, { "X-Order-Token": "one" }),
      api.get("orders/doi-4", undefined, { "X-Order-Token": "one" }),
    ]);
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });

  it("adds them to a POST that has no body, and sends neither a body nor a content type", async () => {
    await api.post("orders/doi-5/payment", undefined, { "X-Order-Token": "secret-token" });
    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(url).toBe("/api/v1/orders/doi-5/payment");
    expect(init.method).toBe("POST");
    expect(init.body).toBeUndefined();
    expect(init.headers).toEqual({ Accept: "application/json", "X-Order-Token": "secret-token" });
  });

  it("does not retry a POST that failed", async () => {
    fetchMock.mockImplementation(() => Promise.resolve(new Response(JSON.stringify({ error: "internal" }), { status: 500 })));
    await expect(api.post("orders/doi-6/payment", undefined, { "X-Order-Token": "t" })).rejects.toMatchObject({ status: 500 });
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });
});
