// @vitest-environment jsdom
import { vi } from "vitest";

// The sandbox can run with NODE_ENV=production, whose React has no act().
vi.hoisted(() => {
  vi.stubEnv("NODE_ENV", "test");
});

vi.mock("../lib/mock/more-ways-to-pay-vietqr-transfer-and-momo", async (original) => ({
  ...(await original<typeof import("../lib/mock/more-ways-to-pay-vietqr-transfer-and-momo")>()),
  getTransfer: vi.fn(),
  putPaid: vi.fn(),
}));

import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import AdminOrderPayment from "../components/AdminOrderPayment";
import OrderDetails from "../components/OrderDetails";
import TransferBlock from "../components/TransferBlock";
import { getTransfer, putPaid, standInModules } from "../lib/mock/more-ways-to-pay-vietqr-transfer-and-momo";
import { payByTransfer } from "../lib/more-ways-to-pay-vietqr-transfer-and-momo";
import { canPay, type PublicOrder } from "../lib/order-page-and-pay-now";

const details = {
  holder: "NGUYEN VAN AN", bank: "Vietcombank", account_number: "0071001234567", amount: 3140000, note: "DOI-24818", payload: "0002010102",
};
const order: PublicOrder = {
  id: "doi-24818", number: "DOI-24818", placed_at: "2026-11-25T10:00:00+07:00", status: "placed", items: [], total: 3140000,
  payment: "bank", paid: false, delivery: "standard", carrier: "", tracking: "", eta: "", customer: "Lan Nguyen", city: "Da Nang",
};
const send = (body: unknown, status = 200) => vi.mocked(getTransfer).mockResolvedValue({ status, body });

beforeEach(() => {
  vi.mocked(getTransfer).mockReset();
  vi.mocked(putPaid).mockReset();
});
afterEach(() => cleanup());

describe("the bank-transfer block", () => {
  it("draws the QR code and the account written out, with the line under them", async () => {
    send(details);
    render(<TransferBlock order={order} />);
    expect(await screen.findByTestId("transfer-block")).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Pay by bank transfer" })).toBeInTheDocument();
    expect(screen.getByRole("img", { name: "QR code to pay order DOI-24818 by bank transfer" })).toBe(screen.getByTestId("transfer-qr"));
    expect(screen.getByTestId("transfer-holder")).toHaveTextContent("NGUYEN VAN AN");
    expect(screen.getByTestId("transfer-bank")).toHaveTextContent("Vietcombank");
    expect(screen.getByTestId("transfer-account")).toHaveTextContent("0071001234567");
    expect(screen.getByTestId("transfer-amount").textContent).toMatch(/^3\.140\.000\s?₫$/);
    expect(screen.getByTestId("transfer-note")).toHaveTextContent("DOI-24818");
    expect(screen.getByText("Scan this QR code with any banking app. We send the pair as soon as the money lands.")).toBeInTheDocument();
    expect(getTransfer).toHaveBeenCalledWith("DOI-24818", 3140000);
  });

  it("is a quiet busy block of the QR's size while the shop is asked, with no heading and no alert", () => {
    vi.mocked(getTransfer).mockReturnValue(new Promise(() => undefined));
    const { container } = render(<TransferBlock order={order} />);
    expect(container.querySelector("[aria-busy='true']")).not.toBeNull();
    expect(screen.queryByRole("heading")).toBeNull();
    expect(screen.queryByRole("alert")).toBeNull();
    expect(screen.queryByTestId("transfer-block")).toBeNull();
  });

  it.each([
    ["a 500", () => send({ error: "internal" }, 500)],
    ["a 404", () => send({ error: "not_found" }, 404)],
    ["no connection", () => vi.mocked(getTransfer).mockRejectedValue(new TypeError("Failed to fetch"))],
    ["an answer without the details", () => send({ holder: "NGUYEN VAN AN" })],
  ])("says it could not load, in alert style where the block would be, after %s", async (_name, fail) => {
    fail();
    render(<TransferBlock order={order} />);
    const alert = await screen.findByTestId("transfer-failed");
    expect(alert).toHaveAttribute("role", "alert");
    expect(alert).toHaveTextContent("We could not load the bank transfer details. Please refresh the page.");
    expect(screen.queryByTestId("transfer-block")).toBeNull();
    expect(screen.queryByText(/can't find/)).toBeNull();
  });

  it("draws nothing when the account is no longer set (503)", async () => {
    send({ error: "payment_unavailable" }, 503);
    const { container } = render(<TransferBlock order={order} />);
    await act(async () => undefined);
    expect(container).toBeEmptyDOMElement();
  });

  it("draws a picture only from the payload, the same for the same payload", () => {
    const a = standInModules("one");
    expect(a).toHaveLength(33);
    expect(standInModules("one")).toEqual(a);
    expect(standInModules("two")).not.toEqual(a);
  });
});

describe("the order page's choice between the QR block and Pay now", () => {
  const show = (over: Partial<PublicOrder>) =>
    render(
      <MemoryRouter>
        <OrderDetails order={{ ...order, ...over }} method={{ id: "x", name: "Bank transfer (VietQR)" }}
          option={{ id: "standard", name: "Standard", days: "2–4 working days", fee: 0 }} />
      </MemoryRouter>,
    );

  it("knows who is paid by the QR code and who by Pay now", () => {
    expect(payByTransfer(order)).toBe(true);
    expect(payByTransfer({ ...order, paid: true })).toBe(false);
    expect(payByTransfer({ ...order, status: "cancelled" })).toBe(false);
    expect(payByTransfer({ ...order, payment: "vnpay" })).toBe(false);
    expect(canPay(order)).toBe(false);
    for (const payment of ["vnpay", "momo"]) expect(canPay({ ...order, payment })).toBe(true);
    expect(canPay({ ...order, payment: "momo", paid: true })).toBe(false);
    expect(canPay({ ...order, payment: "momo", status: "cancelled" })).toBe(false);
  });

  it("draws the QR block and no Pay now for an unpaid bank-transfer order", async () => {
    send(details);
    show({});
    expect(await screen.findByTestId("transfer-block")).toBeInTheDocument();
    expect(screen.queryByTestId("pay-now")).toBeNull();
  });

  it.each([["vnpay"], ["momo"]])("still draws Pay now, and no block, for an unpaid %s order", (payment) => {
    show({ payment });
    expect(screen.getByTestId("pay-now")).toBeInTheDocument();
    expect(screen.queryByTestId("transfer-block")).toBeNull();
    expect(getTransfer).not.toHaveBeenCalled();
  });

  it("draws neither for a paid or a cancelled bank-transfer order, and says the payment is paid", () => {
    show({ paid: true, status: "confirmed" });
    expect(screen.getByText("Bank transfer (VietQR) · paid")).toBeInTheDocument();
    cleanup();
    show({ status: "cancelled" });
    for (const id of ["pay-now", "transfer-block", "transfer-loading"]) expect(screen.queryByTestId(id)).toBeNull();
    expect(getTransfer).not.toHaveBeenCalled();
  });
});

describe("what the dealer sees of an order's payment", () => {
  const mark = (over: Partial<Parameters<typeof AdminOrderPayment>[0]> = {}) => {
    const onMarked = vi.fn();
    render(
      <MemoryRouter>
        <AdminOrderPayment id="doi-24818" way="bank" paid={false} status="placed" onMarked={onMarked} {...over} />
      </MemoryRouter>,
    );
    return onMarked;
  };

  it("offers Mark as paid beside Awaiting payment on an unpaid bank-transfer order, whatever its status", () => {
    for (const status of ["placed", "confirmed", "packed", "shipped", "delivered", "cancelled"] as const) {
      mark({ status });
      expect(screen.getByText("Awaiting payment")).toBeInTheDocument();
      expect(screen.getByTestId("admin-mark-paid")).toHaveTextContent("Mark as paid");
      cleanup();
    }
  });

  it("offers nothing that changes the payment of any other order", () => {
    for (const over of [{ way: "vnpay" }, { way: "momo" }, { way: "bank", paid: true }]) {
      mark(over);
      expect(screen.queryByTestId("admin-mark-paid")).toBeNull();
      cleanup();
    }
  });

  it("waits disabled saying Saving…, then reads Paid, Saved. and the new status, and the button is gone", async () => {
    let answer: (a: { status: number; body: unknown }) => void = () => undefined;
    vi.mocked(putPaid).mockReturnValue(new Promise((done) => { answer = done; }));
    const onMarked = mark();
    fireEvent.click(screen.getByTestId("admin-mark-paid"));
    expect(screen.getByTestId("admin-mark-paid")).toBeDisabled();
    expect(screen.getByTestId("admin-mark-paid")).toHaveTextContent("Saving…");
    expect(putPaid).toHaveBeenCalledWith("doi-24818", "placed");
    await act(async () => answer({ status: 200, body: { id: "doi-24818", paid: true, status: "confirmed" } }));
    expect(screen.getByText("Paid")).toBeInTheDocument();
    expect(screen.queryByText("Awaiting payment")).toBeNull();
    expect(screen.getByRole("status")).toHaveTextContent("Saved.");
    expect(screen.getByTestId("admin-payment-status")).toHaveTextContent("Status: Confirmed");
    expect(screen.queryByTestId("admin-mark-paid")).toBeNull();
    expect(onMarked).toHaveBeenCalledWith({ paid: true, status: "confirmed" });
  });

  it.each([
    ["a refusal", () => vi.mocked(putPaid).mockResolvedValue({ status: 422, body: {} })],
    ["no connection", () => vi.mocked(putPaid).mockRejectedValue(new TypeError("Failed to fetch"))],
  ])("says it could not save after %s and enables the button again", async (_name, fail) => {
    fail();
    const onMarked = mark();
    await act(async () => { fireEvent.click(screen.getByTestId("admin-mark-paid")); });
    expect(screen.getByRole("alert")).toHaveTextContent("We could not save. Please try again.");
    expect(screen.getByTestId("admin-mark-paid")).toBeEnabled();
    expect(screen.getByText("Awaiting payment")).toBeInTheDocument();
    expect(onMarked).not.toHaveBeenCalled();
  });

  it("says the session ended, with a way to sign in, after a 401 or a 403", async () => {
    for (const status of [401, 403]) {
      vi.mocked(putPaid).mockResolvedValue({ status, body: {} });
      mark();
      await act(async () => { fireEvent.click(screen.getByTestId("admin-mark-paid")); });
      expect(screen.getByRole("alert")).toHaveTextContent("Your session ended. Sign in again.");
      expect(screen.getByRole("link", { name: "Sign in again." })).toHaveAttribute("href", "/admin/sign-in");
      cleanup();
    }
  });

  it("reads the provider's transaction number beside a paid order", () => {
    mark({ way: "momo", paid: true, transactionNo: "3123456789" });
    expect(screen.getByText("MoMo transaction 3123456789")).toBeInTheDocument();
    cleanup();
    mark({ way: "vnpay", paid: true, transactionNo: "14123456" });
    expect(screen.getByText("VNPay transaction 14123456")).toBeInTheDocument();
  });
});
