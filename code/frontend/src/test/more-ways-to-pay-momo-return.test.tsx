// @vitest-environment jsdom
import { vi } from "vitest";

// The sandbox can run with NODE_ENV=production, whose React has no act().
vi.hoisted(() => {
  vi.stubEnv("NODE_ENV", "test");
});

vi.mock("../lib/api", async (importOriginal) => {
  const real = await importOriginal<typeof import("../lib/api")>();
  return { ...real, api: { get: vi.fn(), post: vi.fn(), put: vi.fn(), delete: vi.fn() } };
});

import { act, cleanup, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { afterEach, beforeEach, describe, expect, it, type MockInstance } from "vitest";
import { api, ApiError } from "../lib/api";
import { readReturn } from "../lib/more-ways-to-pay-vietqr-transfer-and-momo";
import { rememberOrder } from "../lib/orders";
import PaymentReturn from "../pages/PaymentReturn";

describe("MoMo's return address", () => {
  it("reads the reference and the result code in MoMo's names, success being 0", () => {
    const momo = (qs: string) => readReturn(new URLSearchParams(qs));
    expect(momo("orderId=DOI-24818-1&resultCode=0&signature=abc")).toEqual({ reference: "DOI-24818-1", code: "00" });
    expect(momo("orderId=DOI-24818-1&resultCode=1006")).toEqual({ reference: "DOI-24818-1", code: "1006" });
    expect(momo("vnp_TxnRef=DOI-24818-2&vnp_ResponseCode=00")).toEqual({ reference: "DOI-24818-2", code: "00" });
    expect(momo("vnp_TxnRef=DOI-24818-2&vnp_ResponseCode=24&orderId=other&resultCode=0")).toEqual({ reference: "DOI-24818-2", code: "24" });
    expect(momo("foo=1")).toEqual({ reference: null, code: null });
  });
});

describe("the result page for a MoMo return", () => {
  const get = api.get as unknown as MockInstance<(path: string) => Promise<unknown>>;
  const shop = (extra: Record<string, unknown> = {}) => ({ id: "doi-30001", number: "DOI-30001", status: "placed", paid: false, payment: "momo", ...extra });

  beforeEach(() => {
    vi.useFakeTimers();
    localStorage.clear();
    rememberOrder({
      order: { id: "doi-30001", number: "DOI-30001", placed_at: "2026-11-26T09:00:00+07:00", status: "placed", paid: false, total: 3140000 },
      token: "token-30001", payment_url: "",
    });
    get.mockReset();
    get.mockImplementation(() => Promise.resolve(shop()));
  });
  afterEach(() => {
    cleanup();
    vi.useRealTimers();
  });

  async function land(search: string) {
    render(<MemoryRouter initialEntries={[`/payment/return${search}`]}><PaymentReturn /></MemoryRouter>);
    await act(async () => { await vi.advanceTimersByTimeAsync(0); });
  }
  const state = () => screen.getByTestId("payment-state");
  const ret = (code: string, ref = "DOI-30001-1") => `?partnerCode=P&orderId=${ref}&resultCode=${code}&message=x&signature=${"a".repeat(64)}`;

  it("is confirming for a success code on an unpaid order, and asks the shop with the held token", async () => {
    await land(ret("0"));
    expect(state()).toHaveAttribute("data-state", "confirming");
    expect(state()).toHaveTextContent("We are confirming your payment…");
    expect(get).toHaveBeenCalledWith("orders/doi-30001", undefined, { "X-Order-Token": "token-30001" });
  });

  it("says paid when the shop says so, whatever the address says", async () => {
    get.mockImplementation(() => Promise.resolve(shop({ paid: true, status: "confirmed" })));
    await land(ret("1006"));
    expect(state()).toHaveAttribute("data-state", "paid");
    expect(state()).toHaveTextContent("Payment received.");
  });

  it("is not completed, with Try again, for any other result code on an unpaid order", async () => {
    await land(ret("1006"));
    expect(state()).toHaveAttribute("data-state", "not-completed");
    expect(screen.getByTestId("payment-try-again")).toBeInTheDocument();
  });

  it("names MoMo, not VNPay, when it has not heard from MoMo after 30 seconds, and offers no second payment", async () => {
    await land(ret("0"));
    await act(async () => { await vi.advanceTimersByTimeAsync(31_000); });
    expect(state()).toHaveAttribute("data-state", "timed-out");
    expect(state()).toHaveTextContent("We have not heard from MoMo yet.");
    expect(screen.getByTestId("payment-line")).toHaveTextContent("once MoMo confirms it. Please do not pay again.");
    expect(document.body.textContent).not.toMatch(/VNPay/);
    expect(screen.queryByTestId("payment-try-again")).toBeNull();
  });

  it("keeps VNPay's own words for an order that pays with VNPay, whatever shape the address has", async () => {
    get.mockImplementation(() => Promise.resolve(shop({ payment: "vnpay" })));
    await land(ret("0"));
    await act(async () => { await vi.advanceTimersByTimeAsync(31_000); });
    expect(state()).toHaveTextContent("We have not heard from VNPay yet.");
  });

  it("draws the one cannot-match view, naming neither provider, for another order and for an address of neither shape", async () => {
    for (const search of [ret("0", "DOI-99999-1"), "?foo=1"]) {
      await land(search);
      expect(state()).toHaveAttribute("data-state", "no-match");
      expect(document.body.textContent).not.toMatch(/MoMo|VNPay/);
      cleanup();
    }
  });

  it("says it could not check, in alert style, when the shop fails", async () => {
    get.mockImplementation(() => Promise.reject(new ApiError(500, "internal", "x")));
    await land(ret("0"));
    expect(state()).toHaveAttribute("data-state", "check-failed");
    expect(screen.getByRole("alert")).toHaveTextContent("We could not check your payment. Please open your order to see its state.");
  });
});
