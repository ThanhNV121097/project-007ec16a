import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, render, screen, within } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";

// The sandbox sets NODE_ENV=production and React.act needs the development build.
vi.hoisted(() => {
  vi.stubEnv("NODE_ENV", "test");
});

import OrderRecord from "../components/OrderRecord";
import OrderStatusSelect from "../components/OrderStatusSelect";
import {
  changes, deleteOrder, isLocked, movesFrom, saveOrder, valuesOf, type DealerOrder, type OrderPatch,
} from "../lib/dealer-handles-orders-status-courier-payment";
import { resetMock, transport } from "../lib/mock/dealer-handles-orders-status-courier-payment";

beforeEach(() => {
  resetMock();
  window.history.replaceState({}, "", "/");
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

const load = (id: string) => transport.read(id);
const send = (id: string, body: Record<string, unknown>) => saveOrder(id, body as unknown as OrderPatch);

async function open(id: string) {
  const order = await load(id);
  const onSaved = vi.fn();
  render(
    <MemoryRouter>
      <OrderRecord order={order} title={order.id} paymentName="VNPay" deliveryName="Standard"
        backTo="/admin/orders" onSaved={onSaved} />
    </MemoryRouter>,
  );
  return { order, onSaved };
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
  it("is only the editable fields that changed, never the whole record", async () => {
    const order = await load("doi-24814");
    const values = { ...valuesOf(order), address: "9 New Street", note: "Ring twice." };
    expect(changes(order, values)).toEqual({ address: "9 New Street", note: "Ring twice." });
    expect(changes(order, valuesOf(order))).toEqual({});
  });

  it("is the note alone for a delivered order", async () => {
    const order = await load("doi-24810");
    const values = { ...valuesOf(order), carrier: "GHTK", note: "Left with the porter." };
    expect(changes(order, values)).toEqual({ note: "Left with the porter." });
  });
});

describe("the shop's judgement, as the mock gives it", () => {
  it("moves a placed order to confirmed and no further in one step", async () => {
    expect(await send("doi-24816", { status: "confirmed" })).toMatchObject({ ok: true, order: { status: "confirmed" } });
    const skipped = await send("doi-24816", { status: "shipped" });
    expect(skipped).toMatchObject({ ok: false, kind: "fields", errors: { status: "An order cannot go from confirmed to shipped." } });
  });

  it("refuses leaving delivered, even with the status it has", async () => {
    expect(await send("doi-24810", { status: "delivered" })).toMatchObject({
      ok: false, errors: { status: "An order cannot go from delivered to delivered." },
    });
  });

  it("never lets the dealer mark an order paid, and applies nothing of a refused body", async () => {
    expect(await send("doi-24816", { paid: true })).toMatchObject({
      ok: false, errors: { paid: "Only the payment provider marks this order paid." },
    });
    expect(await send("doi-24816", { status: "confirmed", total: 1 })).toMatchObject({ ok: false, errors: { total: expect.any(String) } });
    expect((await load("doi-24816")).status).toBe("placed");
  });

  it("names a date, an email and a phone it cannot read", async () => {
    const result = await send("doi-24814", { eta: "26/11/2026", email: "abc", phone: "12345678" });
    expect(result).toMatchObject({ ok: false, kind: "fields" });
    expect(result.ok ? {} : Object.keys((result as { errors: object }).errors)).toEqual(["eta", "email", "phone"]);
  });

  it("keeps a delivered order's courier and customer, and its note stays open", async () => {
    expect(await send("doi-24810", { carrier: "GHN" })).toMatchObject({ ok: false, errors: { carrier: expect.any(String) } });
    expect(await send("doi-24810", { note: "Left with the porter." })).toMatchObject({ ok: true, order: { note: "Left with the porter." } });
  });

  it("says unreachable with no answer, and expired with a 401", async () => {
    window.history.replaceState({}, "", "/?mock=offline");
    expect(await send("doi-24816", {})).toEqual({ ok: false, kind: "unreachable" });
    window.history.replaceState({}, "", "/?mock=expired");
    expect(await send("doi-24816", {})).toEqual({ ok: false, kind: "expired" });
  });

  it("deletes only a cancelled order that is not paid", async () => {
    expect(await deleteOrder("doi-24816")).toEqual({ ok: false, kind: "refused", message: "Cancel the order first." });
    await send("doi-24815", { status: "cancelled" });
    expect(await deleteOrder("doi-24815")).toEqual({
      ok: false, kind: "refused", message: "A paid order cannot be deleted; refund it in VNPay's portal first.",
    });
    expect(await deleteOrder("doi-24811")).toEqual({ ok: true });
    await expect(load("doi-24811")).rejects.toMatchObject({ status: 404, code: "not_found" });
  });
});

describe("the order page", () => {
  it("keeps the design's sentence and picture for one pair", async () => {
    await open("doi-24816");
    expect(screen.getByText(/Trang in Biên Hòa ordered 1 × Handball Spezial Earth Strata, EU 37 1\/3, on /)).toBeInTheDocument();
    expect(screen.getAllByRole("img", { name: "Handball Spezial Earth Strata" })).toHaveLength(1);
  });

  it("says how many pairs and lists each one for an order of several", async () => {
    await open("doi-24815");
    expect(screen.getByText(/^Long in Hà Nội ordered 2 pairs on .* — /)).toBeInTheDocument();
    expect(screen.getByText("EU 37 1/3")).toBeInTheDocument();
    expect(screen.getByText("EU 38")).toBeInTheDocument();
    // The name is written beside each picture, so the picture itself says nothing twice.
    expect(screen.getByText("Samba OG Core Black")).toBeInTheDocument();
    expect(screen.queryByRole("img", { name: "Samba OG Core Black" })).toBeNull();
  });

  it("shows how it was paid as text, with VNPay's number when there is one", async () => {
    await open("doi-24810");
    expect(screen.getByText("Paid")).toBeInTheDocument();
    expect(screen.getByText("VNPay transaction 14123456")).toBeInTheDocument();
    expect(screen.queryAllByRole("textbox")).toHaveLength(0);
    cleanup();
    await open("doi-24816");
    expect(screen.getByText("Awaiting payment")).toBeInTheDocument();
    expect(screen.queryByText(/VNPay transaction/)).toBeNull();
  });

  it("draws the status as a control only while the order can move", async () => {
    await open("doi-24816");
    expect(screen.getByRole("combobox", { name: "Status" })).toBeInTheDocument();
    cleanup();
    await open("doi-24810");
    expect(screen.queryByRole("combobox")).toBeNull();
  });

  it("offers the three tabs, with the hint beside Expected", async () => {
    await open("doi-24814");
    expect(screen.getAllByRole("tab").map((t) => t.textContent)).toEqual(["Order", "Delivery", "Customer"]);
    tab("Delivery");
    expect(screen.getByLabelText("Carrier")).toHaveValue("GHTK");
    const hint = screen.getByText("Year-month-day, for example 2026-11-26");
    expect(screen.getByLabelText("Expected")).toHaveAttribute("aria-describedby", hint.id);
  });

  it("saves the address and the note, and says so", async () => {
    const { onSaved } = await open("doi-24814");
    tab("Customer");
    type("Address", "9 New Street");
    type("Internal note", "Ring twice.");
    save();
    expect(await screen.findByText("Saved.")).toBeInTheDocument();
    expect(onSaved).toHaveBeenCalledWith(expect.objectContaining({ address: "9 New Street", note: "Ring twice." }));
  });

  it("names the field the shop refused and keeps every typed value", async () => {
    await open("doi-24814");
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
    await open("doi-24814");
    tab("Delivery");
    type("Expected", "26/11/2026");
    save();
    const sentence = await screen.findByRole("alert");
    const hint = screen.getByText("Year-month-day, for example 2026-11-26");
    expect(screen.getByLabelText("Expected")).toHaveAttribute("aria-invalid", "true");
    expect(screen.getByLabelText("Expected")).toHaveAttribute("aria-describedby", `${hint.id} ${sentence.id}`);
    expect(screen.getByLabelText("Expected")).toHaveValue("26/11/2026");
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
    await open("doi-24814");
    window.history.replaceState({}, "", "/?mock=slow");
    tab("Customer");
    type("Address", "9 New Street");
    save();
    expect(screen.getByRole("button", { name: "Saving…" })).toBeDisabled();
    expect(await screen.findByText("Saved.", {}, { timeout: 4000 })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Save changes" })).toBeEnabled();
  }, 8000);

  it("keeps the order's pairs when the shop's answer comes without them", async () => {
    const { order, onSaved } = await open("doi-24815");
    const answer = Object.entries({ ...order, address: "9 New Street" }).filter(([key]) => key !== "items");
    const update = vi.spyOn(transport, "update").mockResolvedValueOnce(Object.fromEntries(answer) as unknown as DealerOrder);
    tab("Customer");
    type("Address", "9 New Street");
    save();
    expect(await screen.findByText("Saved.")).toBeInTheDocument();
    expect(onSaved).toHaveBeenCalledWith(expect.objectContaining({ address: "9 New Street", items: order.items }));
    update.mockRestore();
  });

  it("says it could not save when the shop does not answer, and keeps the button", async () => {
    window.history.replaceState({}, "", "/?mock=offline");
    await open("doi-24814");
    tab("Customer");
    type("Address", "9 New Street");
    save();
    expect(await screen.findByText("We could not save. Please try again.")).toBeInTheDocument();
    expect(screen.getByLabelText("Address")).toHaveValue("9 New Street");
    expect(screen.getByRole("button", { name: "Save changes" })).toBeEnabled();
  });

  it("links to sign-in when the session has ended", async () => {
    window.history.replaceState({}, "", "/?mock=expired");
    await open("doi-24814");
    save();
    const link = await screen.findByRole("link", { name: "Sign in again." });
    expect(link).toHaveAttribute("href", "/admin/sign-in");
  });

  it("leaves only the note editable on a delivered order", async () => {
    await open("doi-24810");
    tab("Customer");
    expect(screen.getAllByRole("textbox")).toHaveLength(1);
    expect(screen.getByLabelText("Internal note")).toBeEnabled();
  });

  it("asks before deleting and keeps the question open with the shop's sentence", async () => {
    await open("doi-24816");
    fireEvent.click(screen.getByRole("button", { name: "Delete" }));
    const question = screen.getByRole("alertdialog");
    expect(within(question).getByText("Delete doi-24816?")).toBeInTheDocument();
    fireEvent.click(within(question).getByRole("button", { name: "Delete" }));
    expect(await within(question).findByText("Cancel the order first.")).toBeInTheDocument();
    expect(screen.getByRole("alertdialog")).toBeInTheDocument();
  });
});
