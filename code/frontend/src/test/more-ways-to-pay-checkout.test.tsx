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
vi.mock("../lib/orders", async (original) => ({
  ...(await original<typeof import("../lib/orders")>()),
  usePaymentOptions: vi.fn(),
  placeOrder: vi.fn(),
  useCurrentPrices: () => new Map<string, number>(),
}));

import { act, cleanup, fireEvent, render, screen, within } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { setCollection } from "../data";
import { api, ApiError } from "../lib/api";
import { addToCart, clearCart } from "../lib/cart";
import { chosenWay, waysOf } from "../lib/more-ways-to-pay-vietqr-transfer-and-momo";
import { placeOrder, usePaymentOptions, type PaymentOption } from "../lib/orders";
import Checkout from "../pages/Checkout";

const way = (id: string, name: string, over: Partial<PaymentOption> = {}): PaymentOption =>
  ({ id, name, note: `${name} note.`, icon: "card", online: true, is_available: true, ...over });
const VNPAY = way("vnpay", "VNPay — ATM, Visa, Mastercard");
const BANK = way("bank", "Bank transfer (VietQR)", { icon: "qr" });
const MOMO = way("momo", "MoMo wallet", { icon: "wallet" });
const UNAVAILABLE = "Online payment is not available right now. Please try again later.";

const offer = (...items: PaymentOption[]) => vi.mocked(usePaymentOptions).mockReturnValue({ items, loading: false });

function open() {
  return render(
    <MemoryRouter initialEntries={["/checkout"]}>
      <Routes>
        <Route path="/checkout" element={<Checkout />} />
        <Route path="/orders/:id" element={<p>the order page</p>} />
      </Routes>
    </MemoryRouter>,
  );
}

const rows = () => Array.from(document.querySelectorAll("[data-testid^='pay-']")).map((el) => el.getAttribute("data-testid"));
const selected = () => Array.from(document.querySelectorAll("[data-selected='true']")).map((el) => el.getAttribute("data-testid"));
const type = (id: string, value: string) => fireEvent.change(document.getElementById(id) as HTMLElement, { target: { value } });

function fillIn() {
  type("checkout-name", "Lan Nguyen");
  type("checkout-phone", "0912345678");
  type("checkout-address", "12 Example Street, Ward 1");
  type("checkout-city", "Da Nang");
}

beforeEach(() => {
  localStorage.clear();
  clearCart();
  addToCart({ id: "samba-og-w-core-black", name: "Samba OG Black Crystal", price: 3140000, variant: "EU 38" });
  setCollection("delivery", [{ id: "standard", name: "Standard", days: "2–4 working days", fee: 0, note: "" }]);
  vi.mocked(api.get).mockImplementation(() => Promise.reject(new ApiError(404, "not_found", "No.")));
  vi.mocked(placeOrder).mockReset();
});

afterEach(() => cleanup());

describe("which way checkout has chosen", () => {
  it("marks the ways the shop refused and picks the first open one", () => {
    const ways = waysOf([VNPAY, BANK, MOMO], ["vnpay"]);
    expect(ways.map((w) => w.is_available)).toEqual([false, true, true]);
    expect(chosenWay(ways, "")?.id).toBe("bank");
    expect(chosenWay(ways, "momo")?.id).toBe("momo");
    expect(chosenWay(ways, "vnpay")?.id).toBe("bank");
    expect(chosenWay(waysOf([VNPAY], ["vnpay"]), "")).toBeUndefined();
  });
});

describe("checkout's step 03", () => {
  it("draws one radio row for each way the shop lists, in its order, the first one chosen", () => {
    offer(VNPAY, BANK, MOMO);
    open();
    expect(screen.getByRole("radiogroup", { name: "Payment" })).toBeInTheDocument();
    expect(rows()).toEqual(["pay-vnpay", "pay-bank", "pay-momo"]);
    expect(selected()).toEqual(["pay-vnpay"]);
    expect(screen.getByTestId("place-order")).toBeEnabled();
    expect(screen.queryByTestId("pay-cod")).toBeNull();
  });

  it("chooses another row alone, and the summary names it", () => {
    offer(VNPAY, BANK, MOMO);
    open();
    fireEvent.click(screen.getByTestId("pay-bank"));
    expect(selected()).toEqual(["pay-bank"]);
    expect(screen.getByText(/you pay on the Bank transfer \(VietQR\) page/)).toBeInTheDocument();
    fireEvent.click(screen.getByTestId("pay-momo"));
    expect(selected()).toEqual(["pay-momo"]);
  });

  it("is VNPay alone, selected, when nothing else is set up", () => {
    offer(VNPAY);
    open();
    expect(rows()).toEqual(["pay-vnpay"]);
    expect(selected()).toEqual(["pay-vnpay"]);
  });

  it("draws a closed way with the alert under it, cannot choose it, and chooses the next open one", () => {
    offer({ ...VNPAY, is_available: false }, BANK);
    open();
    const closed = screen.getByTestId("pay-vnpay");
    expect(closed).toHaveAttribute("aria-disabled", "true");
    expect(within(closed.parentElement as HTMLElement).getByRole("alert")).toHaveTextContent(UNAVAILABLE);
    expect(selected()).toEqual(["pay-bank"]);
    fireEvent.click(closed);
    expect(selected()).toEqual(["pay-bank"]);
    expect(screen.getByTestId("place-order")).toBeEnabled();
  });

  it("turns Place order off, and says why, when the only way is closed", () => {
    offer({ ...VNPAY, is_available: false });
    open();
    expect(selected()).toEqual([]);
    expect(screen.getByRole("alert")).toHaveTextContent(UNAVAILABLE);
    expect(screen.getByTestId("place-order")).toBeDisabled();
  });

  it("sends the way that is chosen and goes to the order page when the shop has no payment address", async () => {
    offer(VNPAY, BANK, MOMO);
    vi.mocked(placeOrder).mockResolvedValue({
      ok: true,
      placed: {
        order: { id: "doi-24818", number: "DOI-24818", placed_at: "2026-11-26T09:00:00+07:00", status: "placed", paid: false, total: 3140000 },
        token: "token", payment_url: "",
      },
    });
    open();
    fireEvent.click(screen.getByTestId("pay-bank"));
    fillIn();
    await act(async () => { fireEvent.click(screen.getByTestId("place-order")); });
    expect(placeOrder).toHaveBeenCalledTimes(1);
    expect(vi.mocked(placeOrder).mock.calls[0][0].payment).toBe("bank");
    expect(await screen.findByText("the order page")).toBeInTheDocument();
  });

  it("closes a way the shop refuses to place an order with and moves to the next open one", async () => {
    offer(VNPAY, BANK);
    vi.mocked(placeOrder).mockResolvedValue({ ok: false, kind: "unavailable", problems: [] });
    open();
    fillIn();
    await act(async () => { fireEvent.click(screen.getByTestId("place-order")); });
    expect(vi.mocked(placeOrder).mock.calls[0][0].payment).toBe("vnpay");
    expect(screen.getByTestId("pay-vnpay")).toHaveAttribute("aria-disabled", "true");
    expect(selected()).toEqual(["pay-bank"]);
    expect(screen.getByTestId("place-order")).toBeEnabled();
  });
});
