import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

// The sandbox may run with NODE_ENV=production, whose React has no act().
vi.hoisted(() => {
  vi.stubEnv("NODE_ENV", "test");
});

vi.mock("../lib/api", async (original) => ({
  ...(await original<typeof import("../lib/api")>()),
  api: { get: vi.fn(), post: vi.fn(), put: vi.fn(), delete: vi.fn() },
}));
vi.mock("../lib/order-page-and-pay-now", async (original) => ({
  ...(await original<typeof import("../lib/order-page-and-pay-now")>()),
  openPayment: vi.fn(),
}));

import { act, cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import Order from "../pages/Order";
import { ApiError, api } from "../lib/api";
import { canPay, openPayment, requestPayment } from "../lib/order-page-and-pay-now";
import { holdToken, serveOrder } from "./order-fixtures";

const TOKEN = "q3J0n8Zk2LmPa7Vx9TrBwE";
const post = vi.mocked(api.post);

async function showPay(over: Record<string, unknown> = {}) {
  holdToken("doi-24818", TOKEN);
  serveOrder(over);
  render(
    <MemoryRouter initialEntries={["/orders/doi-24818"]}>
      <Routes><Route path="/orders/:id" element={<Order />} /></Routes>
    </MemoryRouter>,
  );
  await screen.findByTestId("order-status");
}

beforeEach(() => {
  localStorage.clear();
  vi.mocked(api.get).mockReset();
  post.mockReset();
  vi.mocked(openPayment).mockReset();
});

afterEach(cleanup);

describe("when Pay now is offered", () => {
  it.each(["placed", "confirmed", "packed", "shipped"])("for an unpaid %s order", async (status) => {
    await showPay({ status });
    expect(screen.getByTestId("pay-now")).toHaveTextContent(/pay now/i);
    expect(screen.getByTestId("pay-now")).toBeEnabled();
  });

  it.each([
    ["a paid order", { paid: true }],
    ["a paid order that is on its way", { paid: true, status: "shipped" }],
    ["a cancelled unpaid order", { status: "cancelled" }],
    ["a cancelled paid order", { status: "cancelled", paid: true }],
  ])("not for %s", async (_name, over) => {
    await showPay(over);
    expect(screen.queryByTestId("pay-now")).toBeNull();
  });

  it("follows the order's state only", () => {
    expect(canPay({ paid: false, status: "placed" })).toBe(true);
    expect(canPay({ paid: true, status: "placed" })).toBe(false);
    expect(canPay({ paid: false, status: "cancelled" })).toBe(false);
  });
});

describe("pressing Pay now", () => {
  it("sends one POST with the token and no body, shows the failure line under the button, and stays on the order", async () => {
    post.mockRejectedValue(new ApiError(409, "conflict", "Online payment is not available yet."));
    await showPay();
    fireEvent.click(screen.getByTestId("pay-now"));
    const line = await screen.findByTestId("pay-now-error");
    expect(line).toHaveTextContent("We could not open the payment page. Please try again.");
    expect(line).toHaveAttribute("role", "alert");
    expect(post).toHaveBeenCalledTimes(1);
    expect(post).toHaveBeenCalledWith("orders/doi-24818/payment", undefined, { "X-Order-Token": TOKEN });
    expect(openPayment).not.toHaveBeenCalled();
    expect(screen.getByTestId("pay-now")).toBeEnabled();
    expect(screen.getByTestId("pay-now")).toHaveTextContent(/pay now/i);
    expect(screen.getByTestId("order-status")).toBeInTheDocument();
  });

  it("clears the failure line when the next press starts", async () => {
    post.mockRejectedValueOnce(new ApiError(409, "conflict", "x"));
    await showPay();
    fireEvent.click(screen.getByTestId("pay-now"));
    await screen.findByTestId("pay-now-error");
    post.mockReturnValueOnce(new Promise(() => {}));
    fireEvent.click(screen.getByTestId("pay-now"));
    expect(screen.queryByTestId("pay-now-error")).toBeNull();
  });

  it.each([
    ["a 503", () => Promise.reject(new ApiError(503, "payment_unavailable", "x"))],
    ["a 500", () => Promise.reject(new ApiError(500, "internal", "x"))],
    ["a 404", () => Promise.reject(new ApiError(404, "not_found", "x"))],
    ["a network failure", () => Promise.reject(new ApiError(0, "network", "x"))],
    ["a 200 with no address", () => Promise.resolve({})],
    ["a 200 with an empty address", () => Promise.resolve({ payment_url: "" })],
    ["a 200 with an address that is not text", () => Promise.resolve({ payment_url: 5 })],
    ["a 200 with a javascript: address", () => Promise.resolve({ payment_url: "javascript:alert(1)" })],
    ["a 200 with an ftp: address", () => Promise.resolve({ payment_url: "ftp://pay.example.test/x" })],
    ["a 200 with a relative address", () => Promise.resolve({ payment_url: "/orders/doi-1" })],
  ])("gives the same failure line and a working button after %s", async (_name, reply) => {
    post.mockImplementation(reply as never);
    await showPay();
    fireEvent.click(screen.getByTestId("pay-now"));
    expect(await screen.findByTestId("pay-now-error")).toHaveTextContent("We could not open the payment page. Please try again.");
    expect(screen.getByTestId("pay-now")).toBeEnabled();
    expect(openPayment).not.toHaveBeenCalled();
  });

  it("goes to exactly the address the shop gives, and presses once while it waits", async () => {
    let reply: (v: unknown) => void = () => {};
    post.mockReturnValue(new Promise((done) => { reply = done; }));
    await showPay();
    const button = screen.getByTestId("pay-now");
    fireEvent.click(button);
    expect(button).toBeDisabled();
    expect(button).toHaveTextContent("Opening the payment page…");
    fireEvent.click(button);
    fireEvent.click(button);
    expect(post).toHaveBeenCalledTimes(1);
    await act(async () => reply({ payment_url: "https://pay.example.test/x" }));
    await waitFor(() => expect(openPayment).toHaveBeenCalledWith("https://pay.example.test/x"));
    expect(openPayment).toHaveBeenCalledTimes(1);
    expect(post).toHaveBeenCalledTimes(1);
  });

  it("works again when the browser comes back to the page with Back", async () => {
    post.mockReturnValue(new Promise(() => {}));
    await showPay();
    fireEvent.click(screen.getByTestId("pay-now"));
    expect(screen.getByTestId("pay-now")).toBeDisabled();
    act(() => {
      window.dispatchEvent(Object.assign(new Event("pageshow"), { persisted: true }));
    });
    expect(screen.getByTestId("pay-now")).toBeEnabled();
    expect(screen.getByTestId("pay-now")).toHaveTextContent(/pay now/i);
  });

  it("asks without the header when this browser holds no token", async () => {
    post.mockResolvedValue({ payment_url: "https://pay.example.test/x" });
    await expect(requestPayment("doi-24818")).resolves.toEqual({ ok: true, url: "https://pay.example.test/x" });
    expect(post).toHaveBeenCalledWith("orders/doi-24818/payment", undefined, undefined);
  });
});
