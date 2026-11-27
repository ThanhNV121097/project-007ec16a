import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

// The API client's calls, standing in for /v1/orders/{id}. Hoisted so the module
// the page's client imports is replaced before it is first read. The sandbox sets
// NODE_ENV=production and React.act needs the development build, so the stub
// lands before React loads. The real ApiError is kept: the client reads its status.
const stubs = vi.hoisted(() => {
  vi.stubEnv("NODE_ENV", "test");
  return { get: vi.fn(), post: vi.fn(), put: vi.fn(), delete: vi.fn() };
});
vi.mock("../lib/api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../lib/api")>();
  return { ...actual, api: { get: stubs.get, post: stubs.post, put: stubs.put, delete: stubs.delete } };
});

import { cleanup, renderHook, waitFor } from "@testing-library/react";
import { ApiError } from "../lib/api";
import {
  changes, deleteOrder, isLocked, movesFrom, saveOrder, useDealerOrder, valuesOf, type DealerOrder,
} from "../lib/dealer-handles-orders-status-courier-payment";

beforeEach(() => {
  vi.clearAllMocks();
});

afterEach(() => {
  cleanup();
});

/** An order as the dealer view answers it. */
const placed: DealerOrder = {
  id: "doi-24816", number: "DOI-24816", placed_at: "2026-11-25T08:40:00+07:00", status: "placed",
  product: "handball-spezial-w-earth-strata", size: "37 1/3", qty: 1, total: 3290000, delivery_fee: 0,
  payment: "vnpay", paid: false, delivery: "standard", carrier: "", tracking: "", eta: "", customer: "Trang",
  city: "Biên Hòa", phone: "0977 302 118", email: "", address: "7 Example Lane", note: "", transaction_no: "",
  items: [{
    product: "handball-spezial-w-earth-strata", name: "Handball Spezial Earth Strata", colourway: "Earth Strata",
    image: "/images/ref/handball-spezial-w-earth-strata.jpg", size: "37 1/3", qty: 1, unit_price: 3290000,
  }],
};
const delivered: DealerOrder = { ...placed, id: "doi-24810", status: "delivered", paid: true, carrier: "GHN" };

const refused = (status: number, code: string, message: string, errors?: Record<string, string>) =>
  new ApiError(status, code, message, errors);
const noAnswer = () => refused(0, "network", "The server cannot be reached.");

describe("the moves the page offers", () => {
  it("allows one step forward, or cancelling before it ships", () => {
    expect(movesFrom("placed")).toEqual(["confirmed", "cancelled"]);
    expect(movesFrom("confirmed")).toEqual(["packed", "cancelled"]);
    expect(movesFrom("packed")).toEqual(["shipped", "cancelled"]);
    expect(movesFrom("shipped")).toEqual(["delivered"]);
  });

  it("locks a delivered or cancelled order", () => {
    expect(isLocked("delivered")).toBe(true);
    expect(isLocked("cancelled")).toBe(true);
    expect(isLocked("shipped")).toBe(false);
  });
});

describe("what a save carries", () => {
  it("is only the editable fields that changed, never the whole record", () => {
    const values = { ...valuesOf(placed), address: "9 New Street", note: "Ring twice." };
    expect(changes(placed, values)).toEqual({ address: "9 New Street", note: "Ring twice." });
    expect(changes(placed, valuesOf(placed))).toEqual({});
  });

  it("is the note alone for a delivered order", () => {
    const values = { ...valuesOf(delivered), carrier: "GHTK", note: "Left with the porter." };
    expect(changes(delivered, values)).toEqual({ note: "Left with the porter." });
  });
});

describe("reading an order", () => {
  it("asks /v1/orders/{id} and says it is ready", async () => {
    stubs.get.mockResolvedValueOnce(placed);
    const { result } = renderHook(() => useDealerOrder("doi-24816"));
    expect(result.current.load).toEqual({ state: "loading" });
    await waitFor(() => expect(result.current.load).toEqual({ state: "ready", order: placed }));
    expect(stubs.get).toHaveBeenCalledWith("orders/doi-24816");
  });

  it("says missing only for the shop's own not_found", async () => {
    stubs.get.mockRejectedValueOnce(refused(404, "not_found", "Not found."));
    const { result } = renderHook(() => useDealerOrder("nope"));
    await waitFor(() => expect(result.current.load).toEqual({ state: "missing" }));
  });

  it("says failed, never missing, when the shop did not answer or answered 5xx", async () => {
    stubs.get.mockRejectedValueOnce(noAnswer());
    const { result, rerender } = renderHook(({ id }) => useDealerOrder(id), { initialProps: { id: "doi-1" } });
    await waitFor(() => expect(result.current.load).toEqual({ state: "failed" }));
    stubs.get.mockRejectedValueOnce(refused(500, "internal", "Something went wrong."));
    rerender({ id: "doi-2" });
    await waitFor(() => expect(result.current.load).toEqual({ state: "failed" }));
  });

  it("says missing for no id at all, and asks the shop nothing", async () => {
    const { result } = renderHook(() => useDealerOrder(undefined));
    await waitFor(() => expect(result.current.load).toEqual({ state: "missing" }));
    expect(stubs.get).not.toHaveBeenCalled();
  });
});

describe("saving an order", () => {
  it("sends a PUT of the patch alone and returns the dealer view the shop answered", async () => {
    const answer = { ...placed, status: "confirmed" };
    stubs.put.mockResolvedValueOnce(answer);
    expect(await saveOrder("doi-24816", { status: "confirmed" })).toEqual({ ok: true, order: answer });
    expect(stubs.put).toHaveBeenCalledWith("orders/doi-24816", { status: "confirmed" });
  });

  it("carries the shop's 422 sentences by field", async () => {
    const errors = { status: "An order cannot go from cancelled to packed." };
    stubs.put.mockRejectedValueOnce(refused(422, "invalid", "Check the order.", errors));
    expect(await saveOrder("doi-24816", { status: "packed" })).toEqual({ ok: false, kind: "fields", errors });
  });

  it("says expired for a 401, and unreachable for no answer or a server fault", async () => {
    stubs.put.mockRejectedValueOnce(refused(401, "unauthenticated", "Sign in to continue."));
    expect(await saveOrder("doi-24816", {})).toEqual({ ok: false, kind: "expired" });
    stubs.put.mockRejectedValueOnce(noAnswer());
    expect(await saveOrder("doi-24816", {})).toEqual({ ok: false, kind: "unreachable" });
    stubs.put.mockRejectedValueOnce(refused(500, "internal", "Something went wrong."));
    expect(await saveOrder("doi-24816", {})).toEqual({ ok: false, kind: "unreachable" });
  });
});

describe("deleting an order", () => {
  it("sends a DELETE and says it is done", async () => {
    stubs.delete.mockResolvedValueOnce(undefined);
    expect(await deleteOrder("doi-24811")).toEqual({ ok: true });
    expect(stubs.delete).toHaveBeenCalledWith("orders/doi-24811");
  });

  it("hands the shop's refusal to the open question", async () => {
    stubs.delete.mockRejectedValueOnce(refused(409, "conflict", "Cancel the order first."));
    expect(await deleteOrder("doi-24816")).toEqual({ ok: false, kind: "refused", message: "Cancel the order first." });
    const paid = "A paid order cannot be deleted; refund it in VNPay's portal first.";
    stubs.delete.mockRejectedValueOnce(refused(409, "conflict", paid));
    expect(await deleteOrder("doi-24815")).toEqual({ ok: false, kind: "refused", message: paid });
  });

  it("says expired for a 401, and unreachable when the shop does not answer", async () => {
    stubs.delete.mockRejectedValueOnce(refused(401, "unauthenticated", "Sign in to continue."));
    expect(await deleteOrder("doi-24815")).toEqual({ ok: false, kind: "expired" });
    stubs.delete.mockRejectedValueOnce(noAnswer());
    expect(await deleteOrder("doi-24815")).toEqual({ ok: false, kind: "unreachable" });
  });
});
