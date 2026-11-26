import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

// The sandbox may run with NODE_ENV=production, whose React has no act().
vi.hoisted(() => {
  vi.stubEnv("NODE_ENV", "test");
});

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

function show(path = "/orders/doi-24818") {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <Routes><Route path="/orders/:id" element={<Order />} /></Routes>
    </MemoryRouter>,
  );
}

const heading = async () => (await screen.findByTestId("order-status")).textContent;
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
    const { container } = show();
    expect(container.querySelector('[aria-busy="true"]')).not.toBeNull();
    expect(screen.queryByRole("heading")).toBeNull();
    expect(screen.queryByText("We can't find that order.")).toBeNull();
    reply(sample);
    expect(await heading()).toBe("We have your order.");
    expect(container.querySelector('[aria-busy="true"]')).toBeNull();
  });

  it("shows one not-found view, with a way to track, only after the shop's own 404, asking without a token", async () => {
    serve(() => Promise.reject(new ApiError(404, "not_found", "Not found.")));
    show("/orders/doi-99999");
    expect(screen.queryByTestId("order-missing")).toBeNull();
    const view = await screen.findByTestId("order-missing");
    expect(view).toHaveTextContent("We can't find that order.");
    expect(view).toHaveTextContent("Check the number in your confirmation message, then look it up with the phone you ordered with.");
    expect(screen.getByRole("link", { name: /Track an order/ })).toHaveAttribute("href", "/track");
    expect(screen.queryByTestId("pay-now")).toBeNull();
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
    show();
    const line = await screen.findByTestId("order-load-error");
    expect(line).toHaveTextContent("We could not load your order. Please try again.");
    expect(line).toHaveAttribute("role", "alert");
    expect(screen.queryByTestId("order-missing")).toBeNull();
    expect(screen.queryByTestId("order-status")).toBeNull();
    serveOrder();
    fireEvent.click(screen.getByRole("button", { name: "Try again" }));
    expect(await heading()).toBe("We have your order.");
    expect(orderReads()).toHaveLength(2);
    expect(screen.queryByTestId("order-load-error")).toBeNull();
  });
});

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
    ["placed", "We have your order."],
    ["confirmed", "Your order is confirmed."],
    ["packed", "Your order is packed."],
    ["shipped", "Your order is on its way."],
    ["delivered", "Your order was delivered."],
    ["cancelled", "This order was cancelled."],
  ])("for status %s reads %s, the one h1", async (value, words) => {
    status(value as PublicOrder["status"]);
    show();
    expect(await heading()).toBe(words);
    expect(screen.getAllByRole("heading", { level: 1 })).toHaveLength(1);
  });

  it("says when a shipped order is expected, and that a delivered one arrived", async () => {
    serveOrder({ status: "shipped", eta: "2026-11-26" });
    const first = show();
    await heading();
    expect(screen.getByText("Expected by 26/11/2026.")).toBeInTheDocument();
    first.unmount();
    serveOrder({ status: "delivered", eta: "2026-11-26" });
    show();
    await heading();
    expect(screen.getByText("Delivered. If your delivery is wrong or damaged, message us.")).toBeInTheDocument();
    expect(screen.queryByText(/Expected by/)).toBeNull();
  });

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
    expect(screen.queryByText(/Expected by/)).toBeNull();
  });

  it("rings the current step of a packed order", async () => {
    status("packed");
    const { container } = show();
    await heading();
    const steps = container.querySelectorAll("ol > li");
    expect(steps).toHaveLength(5);
    expect(container.querySelectorAll('ol > li[aria-current="step"]')).toHaveLength(1);
    expect(steps[2]).toHaveAttribute("aria-current", "step");
  });
});

describe("the pairs and the details", () => {
  it("lists every pair as sold and shows the stored total, not a sum", async () => {
    serveOrder({ items: [pair, { ...kids, unit_price: 2000000 }], total: 7123000 });
    show();
    await heading();
    expect(screen.getByRole("link", { name: "Samba OG Kids Cloud White" })).toHaveAttribute("href", "/shop/samba-og-kids-cloud-white");
    expect(screen.getByText("EU 38 · Qty 1")).toBeInTheDocument();
    expect(screen.getByText("EU 36 · Qty 2")).toBeInTheDocument();
    expect(screen.getByText(/4\.000\.000/)).toBeInTheDocument();
    expect(screen.getByTestId("order-total")).toHaveTextContent(/7\.123\.000/);
  });

  it("links the first pair's picture to its shoe and draws none when it has none", async () => {
    serveOrder();
    const first = show();
    await heading();
    expect(first.container.querySelector('a[aria-label="Samba OG Black Crystal"]')).toHaveAttribute("href", "/shop/samba-og-w-core-black");
    first.unmount();
    serveOrder({ items: [{ ...pair, image: "" }] });
    const bare = show();
    await heading();
    expect(bare.container.querySelector('a[aria-label="Samba OG Black Crystal"]')).toBeNull();
    expect(bare.container.querySelector("img")).toBeNull();
  });

  it("names the delivery and the way to pay from their records, never an id, and says Free", async () => {
    serveOrder();
    const { container } = show();
    await heading();
    expect(screen.getByText("Lan Nguyen, Da Nang")).toBeInTheDocument();
    expect(screen.getByText("Standard · 2–4 working days · Free")).toBeInTheDocument();
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
    const { container } = show();
    await heading();
    const rows = container.querySelectorAll("details");
    expect(rows).toHaveLength(2);
    rows.forEach((row) => expect(row).not.toHaveAttribute("open"));
    expect(rows[0]).toHaveTextContent("Send us the order number and what to change; the address or the size can change until the order is packed.");
    expect(rows[0].querySelector('a[href="/contact"]')).toHaveTextContent("Message us");
    expect(rows[1]).toHaveTextContent("Exchanges are not accepted. For a wrong or damaged delivery, or any question, message us.");
    expect(rows[1].querySelector('a[href="/returns"]')).toHaveTextContent("Return policy");
    expect(screen.getByRole("link", { name: "Keep shopping" })).toHaveAttribute("href", "/shop");
  });
});
