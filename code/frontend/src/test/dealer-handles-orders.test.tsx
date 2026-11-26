import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

// The API client's calls, standing in for /v1/orders/{id}. Hoisted so the module
// the page's client imports is replaced before it is first read. The sandbox sets
// NODE_ENV=production and React.act needs the development build, so the stub
// lands before React loads. The real ApiError is kept: the page reads its status.
const stubs = vi.hoisted(() => {
  vi.stubEnv("NODE_ENV", "test");
  return { get: vi.fn(), post: vi.fn(), put: vi.fn(), delete: vi.fn() };
});
vi.mock("../lib/api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../lib/api")>();
  return { ...actual, api: { get: stubs.get, post: stubs.post, put: stubs.put, delete: stubs.delete } };
});

import { cleanup, fireEvent, render, renderHook, screen, waitFor, within } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import OrderRecord from "../components/OrderRecord";
import OrderStatusSelect from "../components/OrderStatusSelect";
import { ApiError } from "../lib/api";
import {
  changes, deleteOrder, isLocked, movesFrom, saveOrder, useDealerOrder, valuesOf,
  type DealerOrder, type OrderPair,
} from "../lib/dealer-handles-orders-status-courier-payment";

beforeEach(() => {
  vi.clearAllMocks();
});

afterEach(() => {
  cleanup();
});

const strata: OrderPair = {
  product: "handball-spezial-w-earth-strata", name: "Handball Spezial Earth Strata", colourway: "Earth Strata",
  image: "/images/ref/handball-spezial-w-earth-strata.jpg", size: "37 1/3", qty: 1, unit_price: 3290000,
};
const core: OrderPair = {
  product: "samba-og-w-core-black", name: "Samba OG Core Black", colourway: "Core Black",
  image: "/images/ref/samba-og-w-core-black.jpg", size: "38", qty: 1, unit_price: 2600000,
};
const gazelle: OrderPair = {
  product: "gazelle-indoor-solar-gold", name: "Gazelle Indoor Solar Gold", colourway: "Solar Gold",
  image: "/images/ref/gazelle-indoor-solar-gold.jpg", size: "40", qty: 1, unit_price: 2290000,
};

/** An order as the dealer view answers it. */
function order(n: number, f: Partial<DealerOrder> & { items: OrderPair[] }): DealerOrder {
  const [first] = f.items;
  return {
    id: `doi-${n}`, number: `DOI-${n}`, placed_at: "2026-11-23T19:30:00+07:00", status: "placed", product: first.product,
    size: first.size, qty: first.qty, total: f.items.reduce((sum, p) => sum + p.unit_price * p.qty, 0), delivery_fee: 0,
    payment: "vnpay", paid: false, delivery: "standard", carrier: "", tracking: "", eta: "", customer: "", city: "",
    phone: "0905 661 207", email: "", address: "", note: "", transaction_no: "", ...f,
  };
}

const placed = order(24816, { customer: "Trang", city: "Biên Hòa", address: "7 Example Lane", items: [strata] });
const several = order(24815, {
  status: "confirmed", paid: true, transaction_no: "14123457", customer: "Long", city: "Hà Nội", items: [strata, core],
});
const packed = order(24814, {
  status: "packed", carrier: "GHTK", eta: "2026-11-27", customer: "Hương", city: "Vũng Tàu", items: [gazelle],
});
const delivered = order(24810, {
  status: "delivered", paid: true, transaction_no: "14123456", carrier: "GHN", tracking: "GHN-MW55RB",
  customer: "Phương", city: "TP. Hồ Chí Minh", items: [strata],
});

const refused = (status: number, code: string, message: string, errors?: Record<string, string>) =>
  new ApiError(status, code, message, errors);
const noAnswer = () => refused(0, "network", "The server cannot be reached.");

function open(o: DealerOrder) {
  const onSaved = vi.fn();
  render(
    <MemoryRouter>
      <OrderRecord order={o} title={o.id} paymentName="VNPay" deliveryName="Standard"
        backTo="/admin/orders" onSaved={onSaved} />
    </MemoryRouter>,
  );
  return { order: o, onSaved };
}

const tab = (name: string) => fireEvent.mouseDown(screen.getByRole("tab", { name }));
const type = (label: string, value: string) => fireEvent.change(screen.getByLabelText(label), { target: { value } });
const save = () => fireEvent.click(screen.getByRole("button", { name: "Save changes" }));

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
    const values = { ...valuesOf(packed), address: "9 New Street", note: "Ring twice." };
    expect(changes(packed, values)).toEqual({ address: "9 New Street", note: "Ring twice." });
    expect(changes(packed, valuesOf(packed))).toEqual({});
  });

  it("is the note alone for a delivered order", () => {
    const values = { ...valuesOf(delivered), carrier: "GHTK", note: "Left with the porter." };
    expect(changes(delivered, values)).toEqual({ note: "Left with the porter." });
  });
});

describe("the calls to the shop", () => {
  it("reads an order from /v1/orders/{id} and says it is ready", async () => {
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

  it("saves with a PUT of the patch alone and returns the dealer view the shop answered", async () => {
    const answer = { ...placed, status: "confirmed" };
    stubs.put.mockResolvedValueOnce(answer);
    expect(await saveOrder("doi-24816", { status: "confirmed" })).toEqual({ ok: true, order: answer });
    expect(stubs.put).toHaveBeenCalledWith("orders/doi-24816", { status: "confirmed" });
  });

  it("carries the shop's 422 sentences by field, and keeps the typed values to the page", async () => {
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

  it("deletes with a DELETE and hands the shop's refusal to the open question", async () => {
    stubs.delete.mockResolvedValueOnce(undefined);
    expect(await deleteOrder("doi-24811")).toEqual({ ok: true });
    expect(stubs.delete).toHaveBeenCalledWith("orders/doi-24811");
    stubs.delete.mockRejectedValueOnce(refused(409, "conflict", "Cancel the order first."));
    expect(await deleteOrder("doi-24816")).toEqual({ ok: false, kind: "refused", message: "Cancel the order first." });
    const paid = "A paid order cannot be deleted; refund it in VNPay's portal first.";
    stubs.delete.mockRejectedValueOnce(refused(409, "conflict", paid));
    expect(await deleteOrder("doi-24815")).toEqual({ ok: false, kind: "refused", message: paid });
    stubs.delete.mockRejectedValueOnce(refused(401, "unauthenticated", "Sign in to continue."));
    expect(await deleteOrder("doi-24815")).toEqual({ ok: false, kind: "expired" });
    stubs.delete.mockRejectedValueOnce(noAnswer());
    expect(await deleteOrder("doi-24815")).toEqual({ ok: false, kind: "unreachable" });
  });
});

describe("the order page", () => {
  it("keeps the design's sentence and picture for one pair", () => {
    open(placed);
    expect(screen.getByText(/Trang in Biên Hòa ordered 1 × Handball Spezial Earth Strata, EU 37 1\/3, on /)).toBeInTheDocument();
    expect(screen.getAllByRole("img", { name: "Handball Spezial Earth Strata" })).toHaveLength(1);
  });

  it("says how many pairs and lists each one for an order of several", () => {
    open(several);
    expect(screen.getByText(/^Long in Hà Nội ordered 2 pairs on .* — /)).toBeInTheDocument();
    expect(screen.getByText("EU 37 1/3")).toBeInTheDocument();
    expect(screen.getByText("EU 38")).toBeInTheDocument();
    // The name is written beside each picture, so the picture itself says nothing twice.
    expect(screen.getByText("Samba OG Core Black")).toBeInTheDocument();
    expect(screen.queryByRole("img", { name: "Samba OG Core Black" })).toBeNull();
  });

  it("shows how it was paid as text, with VNPay's number when there is one", () => {
    open(delivered);
    expect(screen.getByText("Paid")).toBeInTheDocument();
    expect(screen.getByText("VNPay transaction 14123456")).toBeInTheDocument();
    expect(screen.queryAllByRole("textbox")).toHaveLength(0);
    cleanup();
    open(placed);
    expect(screen.getByText("Awaiting payment")).toBeInTheDocument();
    expect(screen.queryByText(/VNPay transaction/)).toBeNull();
  });

  it("draws the status as a control only while the order can move", () => {
    open(placed);
    expect(screen.getByRole("combobox", { name: "Status" })).toBeInTheDocument();
    cleanup();
    open(delivered);
    expect(screen.queryByRole("combobox")).toBeNull();
  });

  it("offers the three tabs, with the hint beside Expected", () => {
    open(packed);
    expect(screen.getAllByRole("tab").map((t) => t.textContent)).toEqual(["Order", "Delivery", "Customer"]);
    tab("Delivery");
    expect(screen.getByLabelText("Carrier")).toHaveValue("GHTK");
    const hint = screen.getByText("Year-month-day, for example 2026-11-26");
    expect(screen.getByLabelText("Expected")).toHaveAttribute("aria-describedby", hint.id);
  });

  it("saves the address and the note, sends only those two, and says so", async () => {
    stubs.put.mockResolvedValueOnce({ ...packed, address: "9 New Street", note: "Ring twice." });
    const { onSaved } = open(packed);
    tab("Customer");
    type("Address", "9 New Street");
    type("Internal note", "Ring twice.");
    save();
    expect(await screen.findByText("Saved.")).toBeInTheDocument();
    expect(stubs.put).toHaveBeenCalledWith("orders/doi-24814", { address: "9 New Street", note: "Ring twice." });
    expect(onSaved).toHaveBeenCalledWith(expect.objectContaining({ address: "9 New Street", note: "Ring twice." }));
  });

  it("names the field the shop refused and keeps every typed value", async () => {
    stubs.put.mockRejectedValueOnce(
      refused(422, "invalid", "Check the order.", { email: "Enter a valid email address, or leave it empty." }),
    );
    open(packed);
    tab("Customer");
    type("Email", "abc");
    type("City or province", "Hà Nội");
    save();
    const sentence = await screen.findByText("Enter a valid email address, or leave it empty.");
    expect(sentence).toHaveAttribute("role", "alert");
    expect(screen.getByLabelText("Email")).toHaveValue("abc");
    expect(screen.getByLabelText("Email")).toHaveAttribute("aria-invalid", "true");
    expect(screen.getByLabelText("Email")).toHaveAttribute("aria-describedby", sentence.id);
    expect(screen.getByLabelText("City or province")).not.toHaveAttribute("aria-invalid");
    expect(screen.getByLabelText("City or province")).toHaveValue("Hà Nội");
    expect(screen.queryByText("We could not save. Please try again.")).toBeNull();
    expect(screen.getByRole("button", { name: "Save changes" })).toBeEnabled();
  });

  it("ties both the hint and the refusal to Expected", async () => {
    stubs.put.mockRejectedValueOnce(
      refused(422, "invalid", "Check the order.", { eta: "Write the date as year-month-day, for example 2026-11-26." }),
    );
    open(packed);
    tab("Delivery");
    type("Expected", "26/11/2026");
    save();
    const sentence = await screen.findByRole("alert");
    const hint = screen.getByText("Year-month-day, for example 2026-11-26");
    expect(screen.getByLabelText("Expected")).toHaveAttribute("aria-invalid", "true");
    expect(screen.getByLabelText("Expected")).toHaveAttribute("aria-describedby", `${hint.id} ${sentence.id}`);
    expect(screen.getByLabelText("Expected")).toHaveValue("26/11/2026");
  });

  it("shows the shop's sentence for a move it refuses and keeps the typed values", async () => {
    const said = "An order cannot go from cancelled to packed.";
    stubs.put.mockRejectedValueOnce(refused(422, "invalid", "Check the order.", { status: said }));
    open(several);
    tab("Customer");
    type("Address", "9 New Street");
    tab("Order");
    save();
    expect(await screen.findByText(said)).toHaveAttribute("role", "alert");
    expect(screen.queryByText("We could not save. Please try again.")).toBeNull();
    tab("Customer");
    expect(screen.getByLabelText("Address")).toHaveValue("9 New Street");
  });

  it("marks Status invalid and announces the shop's sentence beside it", () => {
    const said = "An order cannot go from cancelled to packed.";
    render(
      <OrderStatusSelect label="Status" value="packed" error={said} onChange={() => undefined}
        options={[{ value: "confirmed", label: "Confirmed" }, { value: "packed", label: "Packed" }]} />,
    );
    const sentence = screen.getByText(said);
    expect(sentence).toHaveAttribute("role", "alert");
    expect(screen.getByRole("combobox", { name: "Status" })).toHaveAttribute("aria-invalid", "true");
    expect(screen.getByRole("combobox", { name: "Status" })).toHaveAttribute("aria-describedby", sentence.id);
  });

  it("reads Saving… on a disabled button while the shop takes its time, then Save changes again", async () => {
    let answer!: (o: DealerOrder) => void;
    stubs.put.mockReturnValueOnce(new Promise<DealerOrder>((resolve) => { answer = resolve; }));
    open(packed);
    tab("Customer");
    type("Address", "9 New Street");
    save();
    expect(screen.getByRole("button", { name: "Saving…" })).toBeDisabled();
    answer({ ...packed, address: "9 New Street" });
    expect(await screen.findByText("Saved.")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Save changes" })).toBeEnabled();
  });

  it("keeps the order's pairs when the shop's answer comes without them", async () => {
    const answer = Object.fromEntries(Object.entries({ ...several, address: "9 New Street" }).filter(([key]) => key !== "items"));
    stubs.put.mockResolvedValueOnce(answer);
    const { onSaved } = open(several);
    tab("Customer");
    type("Address", "9 New Street");
    save();
    expect(await screen.findByText("Saved.")).toBeInTheDocument();
    expect(onSaved).toHaveBeenCalledWith(expect.objectContaining({ address: "9 New Street", items: several.items }));
  });

  it("says it could not save when the shop does not answer, and keeps the button", async () => {
    stubs.put.mockRejectedValueOnce(noAnswer());
    open(packed);
    tab("Customer");
    type("Address", "9 New Street");
    save();
    expect(await screen.findByText("We could not save. Please try again.")).toBeInTheDocument();
    expect(screen.getByLabelText("Address")).toHaveValue("9 New Street");
    expect(screen.getByRole("button", { name: "Save changes" })).toBeEnabled();
  });

  it("links to sign-in when the session has ended", async () => {
    stubs.put.mockRejectedValueOnce(refused(401, "unauthenticated", "Sign in to continue."));
    open(packed);
    tab("Customer");
    type("Address", "9 New Street");
    save();
    const link = await screen.findByRole("link", { name: "Sign in again." });
    expect(link).toHaveAttribute("href", "/admin/sign-in");
    expect(screen.getByLabelText("Address")).toHaveValue("9 New Street");
  });

  it("leaves only the note editable on a delivered order, and sends only the note", async () => {
    stubs.put.mockResolvedValueOnce({ ...delivered, note: "Left with the porter." });
    open(delivered);
    tab("Customer");
    expect(screen.getAllByRole("textbox")).toHaveLength(1);
    expect(screen.getByLabelText("Internal note")).toBeEnabled();
    type("Internal note", "Left with the porter.");
    save();
    expect(await screen.findByText("Saved.")).toBeInTheDocument();
    expect(stubs.put).toHaveBeenCalledWith("orders/doi-24810", { note: "Left with the porter." });
  });

  it("asks before deleting and keeps the question open with the shop's sentence", async () => {
    stubs.delete.mockRejectedValueOnce(refused(409, "conflict", "Cancel the order first."));
    open(placed);
    fireEvent.click(screen.getByRole("button", { name: "Delete" }));
    const question = screen.getByRole("alertdialog");
    expect(within(question).getByText("Delete doi-24816?")).toBeInTheDocument();
    expect(stubs.delete).not.toHaveBeenCalled();
    fireEvent.click(within(question).getByRole("button", { name: "Delete" }));
    expect(await within(question).findByText("Cancel the order first.")).toBeInTheDocument();
    expect(stubs.delete).toHaveBeenCalledWith("orders/doi-24816");
    expect(screen.getByRole("alertdialog")).toBeInTheDocument();
  });

  it("sends no request when the question is cancelled", () => {
    open(placed);
    fireEvent.click(screen.getByRole("button", { name: "Delete" }));
    fireEvent.click(within(screen.getByRole("alertdialog")).getByRole("button", { name: "Cancel" }));
    expect(stubs.delete).not.toHaveBeenCalled();
  });
});
