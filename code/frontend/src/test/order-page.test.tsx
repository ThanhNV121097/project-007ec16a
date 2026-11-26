<<<<<<< refs/heads/feature/order-page-and-pay-now-be
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
=======
import { beforeEach, describe, expect, it, vi } from "vitest";
>>>>>>> refs/heads/main

// The sandbox may run with NODE_ENV=production, whose React has no act().
vi.hoisted(() => {
  vi.stubEnv("NODE_ENV", "test");
});

<<<<<<< refs/heads/feature/order-page-and-pay-now-be
vi.mock("../lib/api", async (original) => ({
  ...(await original<typeof import("../lib/api")>()),
  api: { get: vi.fn(), post: vi.fn(), put: vi.fn(), delete: vi.fn() },
}));

import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import Order from "../pages/Order";
import { ApiError, api } from "../lib/api";
import { vietnamTime, type PublicOrder } from "../lib/order-page-and-pay-now";
import { holdToken, kids, pair, sample, serve, serveOrder } from "./order-fixtures";
=======
vi.mock("../lib/mock/order-page-and-pay-now", () => ({ getOrder: vi.fn(), postPayment: vi.fn() }));
vi.mock("../lib/order-page-and-pay-now", async (original) => ({
  ...(await original<typeof import("../lib/order-page-and-pay-now")>()),
  openPayment: vi.fn(),
}));

import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import Order from "../pages/Order";
import { setCollection } from "../data";
import { openPayment, requestPayment, vietnamTime, type PublicOrder } from "../lib/order-page-and-pay-now";
import { getOrder, postPayment } from "../lib/mock/order-page-and-pay-now";

const pair = { product: "samba-og-w-core-black", name: "Samba OG Black Crystal", colourway: "Core Black", image: "/images/ref/samba-og-w-core-black.jpg", size: "38", qty: 1, unit_price: 3290000 };
const kids = { product: "samba-og-kids-cloud-white", name: "Samba OG Kids Cloud White", colourway: "Cloud White", image: "/images/ref/samba-og-kids-cloud-white.jpg", size: "36", qty: 2, unit_price: 2290000 };

const sample: PublicOrder = {
  id: "doi-24818", number: "DOI-24818", placed_at: "2026-11-25T10:00:00+07:00", status: "placed", items: [pair],
  total: 3290000, payment: "vnpay", paid: false, delivery: "standard", carrier: "", tracking: "", eta: "2026-11-28",
  customer: "Lan Nguyen", city: "Da Nang",
};

const answer = (order: Partial<PublicOrder> & Record<string, unknown> = {}) =>
  vi.mocked(getOrder).mockResolvedValue({ status: 200, body: { ...sample, ...order } });
>>>>>>> refs/heads/main

function show(path = "/orders/doi-24818") {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <Routes><Route path="/orders/:id" element={<Order />} /></Routes>
    </MemoryRouter>,
  );
}

const heading = async () => (await screen.findByTestId("order-status")).textContent;
<<<<<<< refs/heads/feature/order-page-and-pay-now-be
const status = (value: PublicOrder["status"]) => serveOrder({ status: value });
const orderReads = () => vi.mocked(api.get).mock.calls.filter(([path]) => String(path).startsWith("orders/"));

beforeEach(() => {
  localStorage.clear();
  vi.mocked(api.get).mockReset();
});

afterEach(cleanup);

describe("what the page draws before and instead of the order", () => {
  it("draws no heading and no not-found view until the shop has answered, then the order", async () => {
    let reply: (v: unknown) => void = () => {};
    serve(() => new Promise((done) => { reply = done; }));
=======

// The names the page reads from the shop's delivery and payments records (served by the API in the running site).
beforeEach(() => {
  setCollection("delivery", [{ id: "standard", name: "Standard", days: "2–4 working days", fee: 0, note: "" }]);
  setCollection("payments", [{ id: "vnpay", name: "VNPay — ATM, Visa, Mastercard", note: "", online: true }]);
  vi.mocked(getOrder).mockReset();
  vi.mocked(postPayment).mockReset();
  vi.mocked(openPayment).mockReset();
});

describe("what the page draws before and instead of the order", () => {
  it("draws no heading and no not-found view until the shop has answered, then the order", async () => {
    let reply: (v: { status: number; body: unknown }) => void = () => {};
    vi.mocked(getOrder).mockReturnValue(new Promise((done) => { reply = done; }));
>>>>>>> refs/heads/main
    const { container } = show();
    expect(container.querySelector('[aria-busy="true"]')).not.toBeNull();
    expect(screen.queryByRole("heading")).toBeNull();
    expect(screen.queryByText("We can't find that order.")).toBeNull();
<<<<<<< refs/heads/feature/order-page-and-pay-now-be
    reply(sample);
=======
    reply({ status: 200, body: sample });
>>>>>>> refs/heads/main
    expect(await heading()).toBe("We have your order.");
    expect(container.querySelector('[aria-busy="true"]')).toBeNull();
  });

<<<<<<< refs/heads/feature/order-page-and-pay-now-be
  it("shows one not-found view, with a way to track, only after the shop's own 404, asking without a token", async () => {
    serve(() => Promise.reject(new ApiError(404, "not_found", "Not found.")));
    show("/orders/doi-99999");
    expect(screen.queryByTestId("order-missing")).toBeNull();
=======
  it("shows one not-found view, with a way to track, only after a 404", async () => {
    vi.mocked(getOrder).mockResolvedValue({ status: 404, body: { error: "not_found" } });
    show("/orders/doi-99999");
>>>>>>> refs/heads/main
    const view = await screen.findByTestId("order-missing");
    expect(view).toHaveTextContent("We can't find that order.");
    expect(view).toHaveTextContent("Check the number in your confirmation message, then look it up with the phone you ordered with.");
    expect(screen.getByRole("link", { name: /Track an order/ })).toHaveAttribute("href", "/track");
    expect(screen.queryByTestId("pay-now")).toBeNull();
<<<<<<< refs/heads/feature/order-page-and-pay-now-be
    expect(orderReads()).toEqual([["orders/doi-99999", undefined, undefined]]);
  });

  it.each([
    ["a 500", () => Promise.reject(new ApiError(500, "internal", "x"))],
    ["a network failure", () => Promise.reject(new ApiError(0, "network", "x"))],
    ["a 404 that is not the shop's own", () => Promise.reject(new ApiError(404, "internal", "x"))],
    ["a 401", () => Promise.reject(new ApiError(401, "unauthorized", "x"))],
    ["an answer that is not an order", () => Promise.resolve({ ok: true })],
  ])("after %s draws neither the order nor the not-found view, and Try again asks the shop again", async (_name, fail) => {
    serve(fail);
=======
  });

  it.each([
    ["a 500", () => vi.mocked(getOrder).mockResolvedValue({ status: 500, body: {} })],
    ["a network failure", () => vi.mocked(getOrder).mockRejectedValue(new TypeError("Failed to fetch"))],
  ])("after %s draws neither the order nor the not-found view, and Try again asks the shop again", async (_name, fail) => {
    fail();
>>>>>>> refs/heads/main
    show();
    const line = await screen.findByTestId("order-load-error");
    expect(line).toHaveTextContent("We could not load your order. Please try again.");
    expect(line).toHaveAttribute("role", "alert");
    expect(screen.queryByTestId("order-missing")).toBeNull();
    expect(screen.queryByTestId("order-status")).toBeNull();
<<<<<<< refs/heads/feature/order-page-and-pay-now-be
    serveOrder();
    fireEvent.click(screen.getByRole("button", { name: "Try again" }));
    expect(await heading()).toBe("We have your order.");
    expect(orderReads()).toHaveLength(2);
=======
    answer();
    fireEvent.click(screen.getByRole("button", { name: "Try again" }));
    expect(await heading()).toBe("We have your order.");
    expect(getOrder).toHaveBeenCalledTimes(2);
>>>>>>> refs/heads/main
    expect(screen.queryByTestId("order-load-error")).toBeNull();
  });
});

<<<<<<< refs/heads/feature/order-page-and-pay-now-be
describe("the token", () => {
  it("is sent as a header when this browser holds it, and nowhere else", async () => {
    holdToken("doi-24818", "q3J0n8Zk2LmPa7Vx9TrBwE");
    serveOrder();
    const log = vi.spyOn(console, "log");
    const { container } = show("/orders/doi-24818?placed=1");
    await heading();
    expect(orderReads()).toEqual([["orders/doi-24818", undefined, { "X-Order-Token": "q3J0n8Zk2LmPa7Vx9TrBwE" }]]);
    expect(container.innerHTML).not.toContain("q3J0n8Zk2LmPa7Vx9TrBwE");
    expect(log).not.toHaveBeenCalled();
    log.mockRestore();
  });

  it("is the one held for THIS id, never another order's", async () => {
    holdToken("doi-24819", "another-orders-token-xxxxxxxx");
    serveOrder();
    show();
    await heading();
    expect(orderReads()).toEqual([["orders/doi-24818", undefined, undefined]]);
  });

  it("keeps the order after a refresh: the token is read from the store again", async () => {
    holdToken("doi-24818", "q3J0n8Zk2LmPa7Vx9TrBwE");
    serveOrder();
    show("/orders/doi-24818?placed=1").unmount();
    show("/orders/doi-24818?placed=1");
    expect(await heading()).toBe("Thank you — your order is placed.");
  });

  it("draws nothing the shop was not told to send: no phone, address, email, note or token", async () => {
    holdToken("doi-24818", "q3J0n8Zk2LmPa7Vx9TrBwE");
    serveOrder({ phone: "0912345678", address: "12 Example Street", email: "lan@example.com", note: "leave it at the gate", token: "q3J0n8Zk2LmPa7Vx9TrBwE" });
    const { container } = show();
    await heading();
    for (const private_ of ["0912345678", "12 Example Street", "lan@example.com", "leave it at the gate", "q3J0n8Zk2LmPa7Vx9TrBwE"]) {
      expect(container.innerHTML).not.toContain(private_);
    }
  });
});

describe("the heading and the line under it", () => {
  it("is the thanks when the address carries placed, whatever the status, with the placed time in Vietnam's clock", async () => {
    status("delivered");
    show("/orders/doi-24818?placed=1");
    expect(await heading()).toBe("Thank you — your order is placed.");
    expect(screen.getByText(/Order DOI-24818 · placed 25\/11\/2026, 10:00/)).toBeInTheDocument();
    expect(vietnamTime("2026-11-25T10:00:00+07:00")).toBe("2026-11-25T10:00:00");
  });

  it.each([
    ["2026-11-25T05:30:00+07:00", "25/11/2026, 05:30"],
    ["2026-12-31T23:59:00+07:00", "31/12/2026, 23:59"],
  ])("writes the placed time %s as %s: Vietnam's clock, the hour with two digits", async (placedAt, written) => {
    serveOrder({ placed_at: placedAt });
    show();
    await heading();
    expect(screen.getByText(new RegExp(`Order DOI-24818 · placed ${written}`))).toBeInTheDocument();
=======
describe("the heading and the line under it", () => {
  it("is the thanks when the address carries placed, whatever the status", async () => {
    answer({ status: "delivered" });
    show("/orders/doi-24818?placed=1");
    expect(await heading()).toBe("Thank you — your order is placed.");
    expect(screen.getByText(/Order DOI-24818 · placed 25\/11\/2026, 10:00/)).toBeInTheDocument();
>>>>>>> refs/heads/main
  });

  it.each([
    ["placed", "We have your order."],
    ["confirmed", "Your order is confirmed."],
    ["packed", "Your order is packed."],
    ["shipped", "Your order is on its way."],
    ["delivered", "Your order was delivered."],
    ["cancelled", "This order was cancelled."],
<<<<<<< refs/heads/feature/order-page-and-pay-now-be
  ])("for status %s reads %s, the one h1", async (value, words) => {
    status(value as PublicOrder["status"]);
=======
  ])("for status %s reads %s, the one h1", async (status, words) => {
    answer({ status: status as PublicOrder["status"] });
>>>>>>> refs/heads/main
    show();
    expect(await heading()).toBe(words);
    expect(screen.getAllByRole("heading", { level: 1 })).toHaveLength(1);
  });

  it("says when a shipped order is expected, and that a delivered one arrived", async () => {
<<<<<<< refs/heads/feature/order-page-and-pay-now-be
    serveOrder({ status: "shipped", eta: "2026-11-26" });
=======
    answer({ status: "shipped", eta: "2026-11-26" });
>>>>>>> refs/heads/main
    const first = show();
    await heading();
    expect(screen.getByText("Expected by 26/11/2026.")).toBeInTheDocument();
    first.unmount();
<<<<<<< refs/heads/feature/order-page-and-pay-now-be
    serveOrder({ status: "delivered", eta: "2026-11-26" });
=======
    answer({ status: "delivered", eta: "2026-11-26" });
>>>>>>> refs/heads/main
    show();
    await heading();
    expect(screen.getByText("Delivered. If your delivery is wrong or damaged, message us.")).toBeInTheDocument();
    expect(screen.queryByText(/Expected by/)).toBeNull();
  });

<<<<<<< refs/heads/feature/order-page-and-pay-now-be
  it("shows no expected date when the order has none, and the cancelled sentence in place of the steps", async () => {
    serveOrder({ eta: "" });
    const first = show();
    await heading();
    expect(screen.queryByText(/Expected by/)).toBeNull();
    first.unmount();
    serveOrder({ status: "cancelled", eta: "2026-11-26" });
    const { container } = show();
    await heading();
    expect(screen.getByText("This order was cancelled. If you have a question about it, message us.")).toBeInTheDocument();
    expect(container.querySelectorAll("ol > li")).toHaveLength(0);
=======
  it("replaces the five steps with the cancelled sentence and shows no expected date", async () => {
    answer({ status: "cancelled", eta: "2026-11-26" });
    show();
    await heading();
    expect(screen.getByText("This order was cancelled. If you have a question about it, message us.")).toBeInTheDocument();
    expect(screen.queryByRole("list", { name: /./ })).toBeNull();
>>>>>>> refs/heads/main
    expect(screen.queryByText(/Expected by/)).toBeNull();
  });

  it("rings the current step of a packed order", async () => {
<<<<<<< refs/heads/feature/order-page-and-pay-now-be
    status("packed");
=======
    answer({ status: "packed" });
>>>>>>> refs/heads/main
    const { container } = show();
    await heading();
    const steps = container.querySelectorAll("ol > li");
    expect(steps).toHaveLength(5);
    expect(container.querySelectorAll('ol > li[aria-current="step"]')).toHaveLength(1);
    expect(steps[2]).toHaveAttribute("aria-current", "step");
  });
});

describe("the pairs and the details", () => {
<<<<<<< refs/heads/feature/order-page-and-pay-now-be
  it("lists every pair as sold and shows the stored total, not a sum", async () => {
    serveOrder({ items: [pair, { ...kids, unit_price: 2000000 }], total: 7123000 });
=======
  it("lists every pair as sold and shows the stored total", async () => {
    answer({ items: [pair, { ...kids, unit_price: 2000000 }], total: 7123000 });
>>>>>>> refs/heads/main
    show();
    await heading();
    expect(screen.getByRole("link", { name: "Samba OG Kids Cloud White" })).toHaveAttribute("href", "/shop/samba-og-kids-cloud-white");
    expect(screen.getByText("EU 38 · Qty 1")).toBeInTheDocument();
    expect(screen.getByText("EU 36 · Qty 2")).toBeInTheDocument();
    expect(screen.getByText(/4\.000\.000/)).toBeInTheDocument();
    expect(screen.getByTestId("order-total")).toHaveTextContent(/7\.123\.000/);
  });

  it("links the first pair's picture to its shoe and draws none when it has none", async () => {
<<<<<<< refs/heads/feature/order-page-and-pay-now-be
    serveOrder();
=======
    answer();
>>>>>>> refs/heads/main
    const first = show();
    await heading();
    expect(first.container.querySelector('a[aria-label="Samba OG Black Crystal"]')).toHaveAttribute("href", "/shop/samba-og-w-core-black");
    first.unmount();
<<<<<<< refs/heads/feature/order-page-and-pay-now-be
    serveOrder({ items: [{ ...pair, image: "" }] });
    const bare = show();
    await heading();
    expect(bare.container.querySelector('a[aria-label="Samba OG Black Crystal"]')).toBeNull();
    expect(bare.container.querySelector("img")).toBeNull();
  });

  it("names the delivery and the way to pay from their records, never an id, and says Free", async () => {
    serveOrder();
=======
    answer({ items: [{ ...pair, image: "" }] });
    const second = show();
    await heading();
    expect(second.container.querySelector('a[aria-label="Samba OG Black Crystal"]')).toBeNull();
    expect(second.container.querySelector("[data-product-image]")).toBeNull();
    // From 1024px both left-hand blocks start at column 1, so the steps stack under the heading
    // instead of moving up beside it into the space the picture left.
    const blocks = screen.getByTestId("order-status").closest("section")?.children ?? [];
    expect(blocks).toHaveLength(2);
    for (const block of Array.from(blocks)) expect(block).toHaveClass("lg:col-start-1");
  });

  it("names the delivery and the way to pay, never their ids and never 0 ₫", async () => {
    answer({ paid: true });
>>>>>>> refs/heads/main
    const { container } = show();
    await heading();
    expect(screen.getByText("Lan Nguyen, Da Nang")).toBeInTheDocument();
    expect(screen.getByText("Standard · 2–4 working days · Free")).toBeInTheDocument();
<<<<<<< refs/heads/feature/order-page-and-pay-now-be
    expect(screen.getByText("VNPay — ATM, Visa, Mastercard · awaiting payment")).toBeInTheDocument();
    expect(container.textContent).not.toMatch(/\b0 ₫|Express|cash on delivery|to pay on delivery|\bstandard\b|\bvnpay\b/);
    serveOrder({ paid: true });
    cleanup();
    show();
    await heading();
    expect(screen.getByText("VNPay — ATM, Visa, Mastercard · paid")).toBeInTheDocument();
  });

  it("draws the carrier and the tracking number only when they are set", async () => {
    serveOrder({ carrier: "GHN", tracking: "GHN-PT27XM" });
    const first = show();
    await heading();
    expect(screen.getByText("GHN")).toBeInTheDocument();
    expect(screen.getByText("GHN-PT27XM")).toBeInTheDocument();
    first.unmount();
    serveOrder({ carrier: "GHN" });
    const only = show();
    await heading();
    expect(screen.getByText("Carrier")).toBeInTheDocument();
    expect(screen.queryByText("Tracking number")).toBeNull();
    only.unmount();
    serveOrder();
    show();
    await heading();
    expect(screen.queryByText("Carrier")).toBeNull();
    expect(screen.queryByText("Tracking number")).toBeNull();
  });

  it("holds two closed rows with their words and links, and a way back to the shop", async () => {
    serveOrder();
=======
    expect(screen.getByText("VNPay — ATM, Visa, Mastercard · paid")).toBeInTheDocument();
    expect(container.textContent).not.toMatch(/(^|[^\d.,])0\s?₫|\bstandard\b|\bvnpay\b|Express|on delivery/);
  });

  it("draws Carrier and Tracking number only when they are set", async () => {
    answer({ carrier: "GHN", tracking: "GHN-PT27XM" });
    const first = show();
    await heading();
    expect(screen.getByText("GHN-PT27XM")).toBeInTheDocument();
    expect(screen.getByText("Carrier")).toBeInTheDocument();
    first.unmount();
    answer({ carrier: "GHTK" });
    show();
    await heading();
    expect(screen.getByText("Carrier")).toBeInTheDocument();
    expect(screen.queryByText("Tracking number")).toBeNull();
  });

  it("draws nothing of the address, phone, email or note, even if the shop sent them", async () => {
    answer({ phone: "0912345678", email: "lan@example.test", address: "12 Example Street, Ward 1", note: "Gate code 4821" });
    const { container } = show();
    await heading();
    expect(container.innerHTML).not.toMatch(/0912345678|lan@example|12 Example Street|Gate code/);
  });

  it("has two closed rows and a way back to the shop", async () => {
    answer();
>>>>>>> refs/heads/main
    const { container } = show();
    await heading();
    const rows = container.querySelectorAll("details");
    expect(rows).toHaveLength(2);
    rows.forEach((row) => expect(row).not.toHaveAttribute("open"));
<<<<<<< refs/heads/feature/order-page-and-pay-now-be
    expect(rows[0]).toHaveTextContent("Send us the order number and what to change; the address or the size can change until the order is packed.");
    expect(rows[0].querySelector('a[href="/contact"]')).toHaveTextContent("Message us");
    expect(rows[1]).toHaveTextContent("Exchanges are not accepted. For a wrong or damaged delivery, or any question, message us.");
    expect(rows[1].querySelector('a[href="/returns"]')).toHaveTextContent("Return policy");
    expect(screen.getByRole("link", { name: "Keep shopping" })).toHaveAttribute("href", "/shop");
=======
    expect(screen.getByRole("link", { name: "Message us" })).toHaveAttribute("href", "/contact");
    expect(screen.getByRole("link", { name: "Return policy" })).toHaveAttribute("href", "/returns");
    expect(screen.getByRole("link", { name: /Keep shopping/ })).toHaveAttribute("href", "/shop");
  });
});

describe("Pay now", () => {
  it.each([
    ["an unpaid placed order", { status: "placed", paid: false }, true],
    ["an unpaid shipped order", { status: "shipped", paid: false }, true],
    ["a paid order", { status: "placed", paid: true }, false],
    ["an unpaid cancelled order", { status: "cancelled", paid: false }, false],
    ["a paid cancelled order", { status: "cancelled", paid: true }, false],
  ])("is %s: shown = %s", async (_name, order, shown) => {
    answer(order as Partial<PublicOrder>);
    show();
    await heading();
    expect(screen.queryByTestId("pay-now") !== null).toBe(shown);
  });

  it("answers a refusal with one line under the button, and the button works again", async () => {
    answer();
    vi.mocked(postPayment).mockResolvedValue({ status: 409, body: { error: "conflict", message: "Online payment is not available yet." } });
    show();
    await heading();
    fireEvent.click(screen.getByTestId("pay-now"));
    const line = await screen.findByTestId("pay-now-error");
    expect(line).toHaveTextContent("We could not open the payment page. Please try again.");
    expect(line).toHaveAttribute("role", "alert");
    expect(screen.getByTestId("pay-now")).toBeEnabled();
    expect(screen.getByTestId("pay-now")).toHaveTextContent("Pay now");
    expect(openPayment).not.toHaveBeenCalled();
    expect(postPayment).toHaveBeenCalledTimes(1);
    expect(postPayment).toHaveBeenCalledWith("doi-24818");
  });

  it("is disabled and says so while it waits, sends one request, and the failure line goes when the next press starts", async () => {
    answer();
    vi.mocked(postPayment).mockResolvedValueOnce({ status: 503, body: {} });
    show();
    await heading();
    fireEvent.click(screen.getByTestId("pay-now"));
    await screen.findByTestId("pay-now-error");
    let reply: (v: { status: number; body: unknown }) => void = () => {};
    vi.mocked(postPayment).mockReturnValueOnce(new Promise((done) => { reply = done; }));
    fireEvent.click(screen.getByTestId("pay-now"));
    const button = screen.getByTestId("pay-now");
    expect(button).toBeDisabled();
    expect(button).toHaveTextContent("Opening the payment page…");
    expect(screen.queryByTestId("pay-now-error")).toBeNull();
    fireEvent.click(button);
    expect(postPayment).toHaveBeenCalledTimes(2);
    reply({ status: 500, body: {} });
    await waitFor(() => expect(screen.getByTestId("pay-now")).toBeEnabled());
    expect(await screen.findByTestId("pay-now-error")).toBeInTheDocument();
  });

  it("sends the browser to the address the shop gives, and is enabled again when the page comes back", async () => {
    answer();
    vi.mocked(postPayment).mockResolvedValue({ status: 200, body: { payment_url: "https://pay.example.test/x" } });
    show();
    await heading();
    fireEvent.click(screen.getByTestId("pay-now"));
    await waitFor(() => expect(openPayment).toHaveBeenCalledWith("https://pay.example.test/x"));
    expect(screen.getByTestId("pay-now")).toBeDisabled();
    act(() => {
      window.dispatchEvent(Object.assign(new Event("pageshow"), { persisted: true }));
    });
    await waitFor(() => expect(screen.getByTestId("pay-now")).toBeEnabled());
    expect(screen.getByTestId("pay-now")).toHaveTextContent("Pay now");
  });
});

describe("the client", () => {
  it.each([
    ["a 409", { status: 409, body: { error: "conflict" } }],
    ["a 503", { status: 503, body: {} }],
    ["a 500", { status: 500, body: {} }],
    ["a 404", { status: 404, body: {} }],
    ["a 200 with no address", { status: 200, body: {} }],
    ["a 200 with an empty address", { status: 200, body: { payment_url: "" } }],
    ["a 200 with an address that is not http(s)", { status: 200, body: { payment_url: "javascript:alert(1)" } }],
  ])("treats %s as a failure", async (_name, reply) => {
    vi.mocked(postPayment).mockResolvedValue(reply);
    expect(await requestPayment("doi-24818")).toEqual({ ok: false });
  });

  it("treats a network error as a failure and reads an http address as the way on", async () => {
    vi.mocked(postPayment).mockRejectedValueOnce(new TypeError("Failed to fetch"));
    expect(await requestPayment("doi-24818")).toEqual({ ok: false });
    vi.mocked(postPayment).mockResolvedValueOnce({ status: 200, body: { payment_url: "http://pay.example.test/x" } });
    expect(await requestPayment("doi-24818")).toEqual({ ok: true, url: "http://pay.example.test/x" });
  });

  it("writes a time as Vietnam's clock reads it", () => {
    expect(vietnamTime("2026-11-25T03:00:00Z")).toBe("2026-11-25T10:00:00");
    expect(vietnamTime("2026-11-25T10:00:00+07:00")).toBe("2026-11-25T10:00:00");
>>>>>>> refs/heads/main
  });
});
