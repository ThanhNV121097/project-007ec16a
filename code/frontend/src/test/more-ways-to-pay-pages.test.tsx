// @vitest-environment jsdom
import { vi } from "vitest";

// The sandbox can run with NODE_ENV=production, whose React has no act().
vi.hoisted(() => {
  vi.stubEnv("NODE_ENV", "test");
});

// The shop's answers come through the API client; its ApiError stays the real one.
vi.mock("../lib/api", async (original) => ({
  ...(await original<typeof import("../lib/api")>()),
  api: { get: vi.fn(), put: vi.fn(), post: vi.fn(), delete: vi.fn() },
}));

import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import AdminOrderPayment from "../components/AdminOrderPayment";
import OrderDetails from "../components/OrderDetails";
import TransferBlock from "../components/TransferBlock";
import { api, ApiError } from "../lib/api";
import { payByTransfer, qrRows } from "../lib/more-ways-to-pay-vietqr-transfer-and-momo";
import { canPay, type PublicOrder } from "../lib/order-page-and-pay-now";

const get = api.get as unknown as ReturnType<typeof vi.fn>;
const put = api.put as unknown as ReturnType<typeof vi.fn>;

/** A square code of the smallest size, rows of 0 and 1. */
const qr = Array.from({ length: 21 }, (_, r) => Array.from({ length: 21 }, (_, c) => ((r + c) % 3 === 0 ? "1" : "0")).join(""));
const details = {
  holder: "NGUYEN VAN AN", bank: "Vietcombank", account_number: "0071001234567", amount: 3140000, note: "DOI-24818", payload: "0002010102", qr,
};
const order: PublicOrder = {
  id: "doi-24818", number: "DOI-24818", placed_at: "2026-11-25T10:00:00+07:00", status: "placed", items: [], total: 3140000,
  payment: "bank", paid: false, delivery: "standard", carrier: "", tracking: "", eta: "", customer: "Lan Nguyen", city: "Da Nang",
};
const answer = (body: unknown) => get.mockResolvedValue(body);
const refuse = (status: number, code: string) => get.mockRejectedValue(new ApiError(status, code, "The shop said no."));
const hold = (token: string) =>
  localStorage.setItem("doi:placed", JSON.stringify([{ id: "doi-24818", number: "DOI-24818", placed_at: order.placed_at, token }]));

beforeEach(() => {
  get.mockReset();
  put.mockReset();
  localStorage.clear();
});
afterEach(() => cleanup());

describe("the bank-transfer block", () => {
  it("draws the QR code and the account written out, with the line under them", async () => {
    answer(details);
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
  });

  it("draws the rows the shop sent: a 21-row body is a 29-unit picture", async () => {
    answer(details);
    render(<TransferBlock order={order} />);
    const picture = await screen.findByTestId("transfer-qr");
    expect(picture).toHaveAttribute("viewBox", "0 0 29 29");
    expect(picture.querySelector("path")?.getAttribute("d")).toContain("M4 4h1v1h-1z");
  });

  it("asks for the order's transfer with its token in the header, and in no address", async () => {
    hold("tok-secret-1");
    answer(details);
    render(<TransferBlock order={order} />);
    await screen.findByTestId("transfer-block");
    expect(get).toHaveBeenCalledTimes(1);
    expect(get).toHaveBeenCalledWith("orders/doi-24818/transfer", undefined, { "X-Order-Token": "tok-secret-1" });
    expect(String(get.mock.calls[0][0])).not.toContain("tok-secret-1");
  });

  it("sends no header when no token is held, so a signed-in customer's session speaks", async () => {
    answer(details);
    render(<TransferBlock order={order} />);
    await screen.findByTestId("transfer-block");
    expect(get).toHaveBeenCalledWith("orders/doi-24818/transfer", undefined, undefined);
  });

  it("is a quiet busy block of the QR's size while the shop is asked, with no heading and no alert", () => {
    get.mockReturnValue(new Promise(() => undefined));
    const { container } = render(<TransferBlock order={order} />);
    expect(container.querySelector("[aria-busy='true']")).not.toBeNull();
    expect(screen.queryByRole("heading")).toBeNull();
    expect(screen.queryByRole("alert")).toBeNull();
    expect(screen.queryByTestId("transfer-block")).toBeNull();
  });

  it.each([
    ["a 500", () => refuse(500, "internal")],
    ["a 404", () => refuse(404, "not_found")],
    ["no connection", () => refuse(0, "network")],
    ["an answer without the details", () => answer({ holder: "NGUYEN VAN AN" })],
    ["a code whose rows are ragged", () => answer({ ...details, qr: [...qr.slice(0, 20), "0101"] })],
    ["a code of fewer than 21 rows", () => answer({ ...details, qr: qr.slice(0, 5).map((r) => r.slice(0, 5)) })],
    ["a code with a character that is not 0 or 1", () => answer({ ...details, qr: ["x".repeat(21), ...qr.slice(1)] })],
  ])("says it could not load, in alert style where the block would be, after %s", async (_name, fail) => {
    fail();
    render(<TransferBlock order={order} />);
    const alert = await screen.findByTestId("transfer-failed");
    expect(alert).toHaveAttribute("role", "alert");
    expect(alert).toHaveTextContent("We could not load the bank transfer details. Please refresh the page.");
    expect(screen.queryByTestId("transfer-block")).toBeNull();
    expect(screen.queryByText(/can't find/)).toBeNull();
  });

  it.each([[503, "payment_unavailable"], [409, "conflict"]])("draws nothing after a %s: the account is gone or the order is no longer to be paid by transfer", async (status, code) => {
    refuse(status, code);
    const { container } = render(<TransferBlock order={order} />);
    await act(async () => undefined);
    expect(container).toBeEmptyDOMElement();
  });

  it("turns the shop's rows into dark and light modules", () => {
    expect(qrRows(["101", "010"])).toEqual([[true, false, true], [false, true, false]]);
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
    answer(details);
    show({});
    expect(await screen.findByTestId("transfer-block")).toBeInTheDocument();
    expect(screen.queryByTestId("pay-now")).toBeNull();
  });

  it.each([["vnpay"], ["momo"]])("still draws Pay now, and no block, for an unpaid %s order", (payment) => {
    show({ payment });
    expect(screen.getByTestId("pay-now")).toBeInTheDocument();
    expect(screen.queryByTestId("transfer-block")).toBeNull();
    expect(get).not.toHaveBeenCalled();
  });

  it("draws neither for a paid or a cancelled bank-transfer order, and says the payment is paid", () => {
    show({ paid: true, status: "confirmed" });
    expect(screen.getByText("Bank transfer (VietQR) · paid")).toBeInTheDocument();
    cleanup();
    show({ status: "cancelled" });
    for (const id of ["pay-now", "transfer-block", "transfer-loading"]) expect(screen.queryByTestId(id)).toBeNull();
    expect(get).not.toHaveBeenCalled();
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
    let reply: (body: unknown) => void = () => undefined;
    put.mockReturnValue(new Promise((done) => { reply = done; }));
    const onMarked = mark();
    fireEvent.click(screen.getByTestId("admin-mark-paid"));
    expect(screen.getByTestId("admin-mark-paid")).toBeDisabled();
    expect(screen.getByTestId("admin-mark-paid")).toHaveTextContent("Saving…");
    // The body is {"paid": true} and nothing else.
    expect(put).toHaveBeenCalledWith("orders/doi-24818", { paid: true });
    await act(async () => reply({ id: "doi-24818", paid: true, status: "confirmed" }));
    expect(screen.getByText("Paid")).toBeInTheDocument();
    expect(screen.queryByText("Awaiting payment")).toBeNull();
    expect(screen.getByRole("status")).toHaveTextContent("Saved.");
    expect(screen.getByTestId("admin-payment-status")).toHaveTextContent("Status: Confirmed");
    expect(screen.queryByTestId("admin-mark-paid")).toBeNull();
    expect(onMarked).toHaveBeenCalledWith({ paid: true, status: "confirmed" });
  });

  it.each([
    ["a refusal", () => put.mockRejectedValue(new ApiError(422, "invalid", "Check the details of this order.", { paid: "Only the payment provider marks this order paid." }))],
    ["no connection", () => put.mockRejectedValue(new ApiError(0, "network", "The server cannot be reached."))],
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
    for (const [status, code] of [[401, "unauthenticated"], [403, "forbidden"]] as const) {
      put.mockRejectedValue(new ApiError(status, code, "No."));
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
