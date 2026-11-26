// @vitest-environment jsdom
import { vi } from "vitest";

// The sandbox can run with NODE_ENV=production, whose React has no act(): the
// stub has to land before React loads.
vi.hoisted(() => {
  vi.stubEnv("NODE_ENV", "test");
});

// The shop is the API client's: its two calls are stubbed, its real ApiError is kept.
vi.mock("../lib/api", async (importOriginal) => {
  const real = await importOriginal<typeof import("../lib/api")>();
  return { ...real, api: { get: vi.fn(), post: vi.fn(), put: vi.fn(), delete: vi.fn() } };
});

import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { afterEach, beforeEach, describe, expect, it, type MockInstance } from "vitest";
import { api, ApiError } from "../lib/api";
import { decide, GIVE_UP_MS, navigation, orderNumberOf, POLL_MS } from "../lib/online-payment-with-vnpay";
import { rememberOrder } from "../lib/orders";
import PaymentReturn from "../pages/PaymentReturn";

type Call = (path: string, second?: unknown, headers?: Record<string, string>) => Promise<unknown>;
const get = api.get as unknown as MockInstance<Call>;
const post = api.post as unknown as MockInstance<Call>;

const TOKEN = "token-of-doi-30001";
const ID = "doi-30001";
const SANDBOX = "https://sandbox.vnpayment.vn/paymentv2/vpcpay.html?vnp_TxnRef=DOI-30001-2";
const HEADERS = { "X-Order-Token": TOKEN };

const ok = (body: unknown): Promise<unknown> => Promise.resolve(body);
const refuse = (status: number, code: string): Promise<unknown> => Promise.reject(new ApiError(status, code, "The shop said no."));
const offline = (): Promise<unknown> => Promise.reject(new ApiError(0, "network", "The server cannot be reached."));
const order = (extra: { status?: string; paid?: boolean } = {}) => ({ id: ID, number: "DOI-30001", status: "placed", paid: false, ...extra });

let leave: MockInstance<typeof navigation.to>;

function hold(id: string, number: string, token: string) {
  rememberOrder({
    order: { id, number, placed_at: "2026-11-26T09:00:00+07:00", status: "placed", paid: false, total: 3140000 },
    token,
    payment_url: "",
  });
}

beforeEach(() => {
  vi.useFakeTimers();
  localStorage.clear();
  hold(ID, "DOI-30001", TOKEN);
  get.mockReset();
  post.mockReset();
  get.mockImplementation(() => ok(order()));
  post.mockImplementation(() => ok({ payment_url: SANDBOX }));
  leave = vi.spyOn(navigation, "to").mockImplementation(() => undefined);
});

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
  vi.useRealTimers();
});

const elapse = (ms: number) => act(async () => { await vi.advanceTimersByTimeAsync(ms); });

async function open(search: string) {
  render(
    <MemoryRouter initialEntries={[`/payment/return${search}`]}>
      <PaymentReturn />
    </MemoryRouter>,
  );
  await elapse(0);
}

const returned = (code: string, ref = "DOI-30001-1") => `?vnp_TxnRef=${ref}&vnp_ResponseCode=${code}&vnp_SecureHash=${"a".repeat(128)}`;
const heading = () => screen.getByTestId("payment-state");
const line = () => screen.getByTestId("payment-line");
const tryAgain = () => screen.queryByTestId("payment-try-again");

describe("what the address may and may not decide", () => {
  it("takes the order number out of a reference, and nothing else", () => {
    expect(orderNumberOf("DOI-24818-2")).toBe("DOI-24818");
    expect(orderNumberOf("doi-24818-12")).toBe("DOI-24818");
    for (const bad of ["abc", "DOI-24818", "DOI--1", "DOI-24818-1; drop", "", null]) expect(orderNumberOf(bad)).toBeNull();
  });

  it("lets only the shop say paid or cancelled; the code chooses between confirming and not completed", () => {
    const unpaid = order();
    expect(decide(unpaid, "00", false)).toBe("confirming");
    expect(decide(unpaid, "00", true)).toBe("timed-out");
    for (const code of ["24", "07", "", null]) expect(decide(unpaid, code, false)).toBe("not-completed");
    expect(decide(order({ paid: true }), "24", false)).toBe("paid");
    expect(decide(order({ status: "cancelled", paid: true }), "00", false)).toBe("cancelled");
  });
});

describe("the Payment result page", () => {
  it("shows only a quiet placeholder until the shop has answered", async () => {
    let release!: (body: unknown) => void;
    get.mockReturnValue(new Promise<unknown>((resolve) => { release = resolve; }));
    await open(returned("00"));
    expect(document.querySelector("[aria-busy='true']")).not.toBeNull();
    expect(screen.queryByTestId("payment-state")).toBeNull();
    expect(tryAgain()).toBeNull();
    await act(async () => { release(order()); });
    expect(heading()).toHaveAttribute("data-state", "confirming");
    expect(document.querySelector("[aria-busy='true']")).toBeNull();
  });

  it("says paid whatever the address says, with the token the browser holds", async () => {
    get.mockImplementation(() => ok(order({ paid: true, status: "confirmed" })));
    await open(returned("24"));
    expect(get).toHaveBeenCalledWith(`orders/${ID}`, undefined, HEADERS);
    expect(heading().textContent).toBe("Payment received.");
    expect(heading()).toHaveAttribute("data-state", "paid");
    expect(line().textContent).toBe("Your order DOI-30001 is paid. We will pack it next.");
    expect(screen.getByTestId("payment-view-order")).toHaveAttribute("href", `/orders/${ID}`);
    expect(tryAgain()).toBeNull();
  });

  it("confirms every 3 seconds, then says paid in the same status region with no reload", async () => {
    await open(returned("00"));
    expect(heading().textContent).toBe("We are confirming your payment…");
    expect(line().textContent).toBe("This usually takes a few seconds. Please do not pay again.");
    const region = line().closest("[role='status']");
    expect(region).not.toBeNull();
    expect(tryAgain()).toBeNull();
    expect(get).toHaveBeenCalledTimes(1);
    await elapse(POLL_MS);
    expect(get).toHaveBeenCalledTimes(2);
    get.mockImplementation(() => ok(order({ paid: true, status: "confirmed" })));
    await elapse(POLL_MS);
    expect(heading()).toHaveAttribute("data-state", "paid");
    expect(region).toBe(line().closest("[role='status']"));
    expect(region).toHaveTextContent("Your order DOI-30001 is paid.");
  });

  it("gives up after 30 seconds, stops asking and offers no second payment", async () => {
    await open(returned("00"));
    await elapse(GIVE_UP_MS + 1_000);
    expect(heading().textContent).toBe("We have not heard from VNPay yet.");
    expect(heading()).toHaveAttribute("data-state", "timed-out");
    expect(line().textContent).toBe("If money left your account it will show on your order once VNPay confirms it. Please do not pay again.");
    expect(screen.getByTestId("payment-view-order")).toBeInTheDocument();
    expect(tryAgain()).toBeNull();
    const asked = get.mock.calls.length;
    await elapse(15_000);
    expect(get.mock.calls.length).toBe(asked);
  });

  it("offers Try again for an unpaid order the address says was not paid, and sends the browser to the new address", async () => {
    await open(returned("24"));
    expect(heading().textContent).toBe("Payment was not completed.");
    expect(line().textContent).toBe("You can try again, or place a new order.");
    expect(get).toHaveBeenCalledTimes(1);
    fireEvent.click(tryAgain() as HTMLElement);
    expect(tryAgain()).toBeDisabled();
    await elapse(0);
    // One POST, no body, with the order's token in the header and nowhere else.
    expect(post).toHaveBeenCalledTimes(1);
    expect(post).toHaveBeenCalledWith(`orders/${ID}/payment`, undefined, HEADERS);
    expect(leave).toHaveBeenCalledWith(SANDBOX);
  });

  it("keeps the page and says so when the payment page cannot be opened", async () => {
    post.mockImplementation(() => refuse(503, "payment_unavailable"));
    await open(returned("24"));
    fireEvent.click(tryAgain() as HTMLElement);
    await elapse(0);
    expect(screen.getByRole("alert").textContent).toBe("We could not open the payment page. Please try again.");
    expect(tryAgain()).toBeEnabled();
    expect(leave).not.toHaveBeenCalled();
    post.mockImplementation(() => offline());
    fireEvent.click(tryAgain() as HTMLElement);
    await elapse(0);
    expect(screen.getByRole("alert")).toBeInTheDocument();
    expect(leave).not.toHaveBeenCalled();
  });

  it("asks for the order again after a 409 and draws what it now is", async () => {
    post.mockImplementation(() => refuse(409, "conflict"));
    await open(returned("24"));
    get.mockImplementation(() => ok(order({ status: "cancelled" })));
    fireEvent.click(tryAgain() as HTMLElement);
    await elapse(0);
    expect(heading()).toHaveAttribute("data-state", "cancelled");
    expect(leave).not.toHaveBeenCalled();
  });

  it("draws a given-up order as cancelled, even if a payment arrived late", async () => {
    get.mockImplementation(() => ok(order({ status: "cancelled", paid: true })));
    await open(returned("00"));
    expect(heading().textContent).toBe("This order was cancelled.");
    expect(heading()).toHaveAttribute("data-state", "cancelled");
    expect(line().textContent).toBe("The time to pay ran out, so the pairs went back on the shelf. You can order again.");
    expect(screen.getByTestId("payment-keep-shopping")).toHaveAttribute("href", "/shop");
    expect(tryAgain()).toBeNull();
  });

  it("shows one same view, and asks nothing, for a payment this browser holds no token for", async () => {
    for (const search of [returned("00", "DOI-99999-1"), "?vnp_ResponseCode=00", returned("00", "abc")]) {
      await open(search);
      expect(heading().textContent).toBe("We can't match this payment to an order on this browser.");
      expect(heading()).toHaveAttribute("data-state", "no-match");
      expect(line().textContent).toBe("Look the order up with its number and phone.");
      expect(screen.getByTestId("payment-track")).toHaveAttribute("href", "/track");
      expect(screen.queryByTestId("payment-view-order")).toBeNull();
      expect(document.body.textContent).not.toMatch(/DOI-\d|paid|₫/);
      cleanup();
    }
    expect(get).not.toHaveBeenCalled();
  });

  it("draws the one cannot-match view when the shop answers not_found for a held token", async () => {
    get.mockImplementation(() => refuse(404, "not_found"));
    await open(returned("00"));
    expect(heading()).toHaveAttribute("data-state", "no-match");
    expect(screen.getByTestId("payment-track")).toHaveAttribute("href", "/track");
    expect(screen.queryByTestId("payment-view-order")).toBeNull();
    expect(document.body.textContent).not.toMatch(/DOI-\d/);
  });

  it("does not call a 404 that is not the shop's own answer a no-match", async () => {
    get.mockImplementation(() => refuse(404, "internal"));
    await open(returned("00"));
    expect(heading()).toHaveAttribute("data-state", "check-failed");
    expect(screen.getByTestId("payment-view-order")).toHaveAttribute("href", `/orders/${ID}`);
  });

  it("says it could not check, never a result, when the shop fails", async () => {
    get.mockImplementation(() => refuse(500, "internal"));
    await open(returned("00"));
    expect(heading()).toHaveAttribute("data-state", "check-failed");
    expect(screen.getByRole("alert").textContent).toBe("We could not check your payment. Please open your order to see its state.");
    expect(screen.getByTestId("payment-view-order")).toHaveAttribute("href", `/orders/${ID}`);
    expect(document.body.textContent).not.toMatch(/Payment received|cancelled|can't match/);
    expect(tryAgain()).toBeNull();
    cleanup();
    get.mockImplementation(() => offline());
    await open(returned("00"));
    expect(heading()).toHaveAttribute("data-state", "check-failed");
    cleanup();
    get.mockImplementation(() => ok({ not: "an order" }));
    await open(returned("00"));
    expect(heading()).toHaveAttribute("data-state", "check-failed");
  });

  it("keeps what it knew when a later ask fails while confirming", async () => {
    await open(returned("00"));
    get.mockImplementation(() => refuse(500, "internal"));
    await elapse(POLL_MS);
    expect(heading()).toHaveAttribute("data-state", "confirming");
  });

  it("sends only the token of the order the reference names, in the header and never in an address", async () => {
    hold("doi-30002", "DOI-30002", "token-of-doi-30002");
    await open(returned("00"));
    await elapse(POLL_MS);
    expect(get.mock.calls.length).toBeGreaterThan(1);
    for (const [path, params, headers] of get.mock.calls) {
      expect(path).toBe(`orders/${ID}`);
      expect(params).toBeUndefined();
      expect(headers).toEqual(HEADERS);
    }
    expect(JSON.stringify([...get.mock.calls, ...post.mock.calls])).not.toContain("token-of-doi-30002");
    expect(JSON.stringify(leave.mock.calls)).not.toContain(TOKEN);
  });

  it("has one h1 and every word from the content", async () => {
    await open(returned("24"));
    expect(document.querySelectorAll("h1")).toHaveLength(1);
    expect(heading().getAttribute("data-edit")).toBe("pages.payment_return.not_completed.heading");
  });
});
