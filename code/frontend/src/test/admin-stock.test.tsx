import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";

// The sandbox sets NODE_ENV=production and React.act needs the development build.
vi.hoisted(() => {
  vi.stubEnv("NODE_ENV", "test");
});

vi.mock("../lib/mock/manage-shoes-lines-categories-reviews-and-orders", () => ({
  default: { create: vi.fn(), update: vi.fn(), remove: vi.fn(), read: vi.fn(), upload: vi.fn(), setPairs: vi.fn() },
}));

import RecordForm from "../components/admin/RecordForm";
import { fieldsOf } from "../components/admin/fields";
import type { Row } from "../data";
import transport from "../lib/mock/manage-shoes-lines-categories-reviews-and-orders";
import type { Wire } from "../lib/manage-shoes-lines-categories-reviews-and-orders";

const shop = vi.mocked(transport);
const reply = (status: number, body?: unknown): Wire => ({ status, body });
const TAKEN = "Pairs were just taken by an order. Check the number and save again.";

beforeEach(() => {
  Object.values(shop).forEach((fn) => fn.mockReset());
});
afterEach(cleanup);

const shoe: Row = {
  id: "tc-shoe", name: "Tc Shoe", price: 3290000, status: "live",
  sizes: ["38", "39", "40"], sizes_low: ["39"], sizes_out: ["40"], stock: 6,
  pairs: { "38": 4, "39": 2, "40": 0 },
};
const only = (...keys: string[]) => fieldsOf("products").filter((f) => keys.includes(f.key));

function open(keys: string[]) {
  render(
    <MemoryRouter>
      <RecordForm collection="products" record={shoe} specs={only(...keys)} title="Shoes" backTo="/admin/products" remove />
    </MemoryRouter>,
  );
}
const count = (size: string) => screen.getByLabelText(`Pairs in EU ${size}`);
const change = (el: HTMLElement, value: string) => fireEvent.change(el, { target: { value } });
const type = (label: string, value: string) => change(screen.getByLabelText(label), value);

describe("the Stock tab of a shoe's page", () => {
  const stockTab = ["sizes", "sizes_low", "sizes_out", "stock"];

  it("has a labelled whole-number count for each size and three lists that are only read", () => {
    open(stockTab);
    for (const [size, pairs] of [["38", 4], ["39", 2], ["40", 0]] as const) {
      const field = count(size);
      expect(field).toHaveValue(pairs);
      expect(field).toHaveAttribute("inputmode", "numeric");
      expect(field).toHaveAttribute("min", "0");
      expect(field).toHaveAttribute("step", "1");
    }
    for (const [label, shown] of [["Last pairs in", "39"], ["Sold out in", "40"], ["Pairs in stock", "6"]]) {
      const value = screen.getByLabelText(label);
      expect(value.tagName).toBe("OUTPUT");
      expect(value).toHaveTextContent(shown);
    }
  });

  it("saves one size by itself on blur, says Saving… then Saved., and redraws the lists from the answer", async () => {
    const answer = new Promise<Wire>((done) => {
      setTimeout(() => done(reply(200, { size: "38", pairs: 9, sizes: ["38", "39", "40"], sizes_low: ["39"], sizes_out: ["40"], stock: 11 })), 0);
    });
    shop.setPairs.mockReturnValueOnce(answer);
    open(stockTab);
    change(count("38"), "9");
    fireEvent.blur(count("38"));
    expect(await screen.findByText("Saving…")).toBeInTheDocument();
    expect(await screen.findByText("Saved.")).toBeInTheDocument();
    expect(shop.setPairs).toHaveBeenCalledTimes(1);
    expect(shop.setPairs).toHaveBeenCalledWith("tc-shoe", { size: "38", pairs: 9, seen_pairs: 4 });
    expect(screen.getByLabelText("Pairs in stock")).toHaveTextContent("11");
  });

  it("saves on Enter, and sends nothing for a number that did not change", async () => {
    shop.setPairs.mockResolvedValueOnce(reply(200, { size: "39", pairs: 4, sizes: ["38", "39", "40"], sizes_low: [], sizes_out: ["40"], stock: 8 }));
    open(stockTab);
    fireEvent.focus(count("40"));
    fireEvent.blur(count("40"));
    change(count("39"), "4");
    fireEvent.keyDown(count("39"), { key: "Enter" });
    expect(await screen.findByText("Saved.")).toBeInTheDocument();
    expect(shop.setPairs).toHaveBeenCalledTimes(1);
    expect(screen.getByLabelText("Last pairs in")).toHaveTextContent("None");
  });

  it("shows the count that stands and the shop's sentence when an order took pairs meanwhile", async () => {
    shop.setPairs.mockResolvedValueOnce(reply(409, { error: "conflict", message: TAKEN }));
    shop.read.mockResolvedValueOnce(reply(200, { ...shoe, pairs: { "38": 2, "39": 2, "40": 0 }, sizes_low: ["38", "39"], stock: 4 }));
    open(stockTab);
    change(count("38"), "6");
    fireEvent.blur(count("38"));
    expect(await screen.findByText(TAKEN)).toBeInTheDocument();
    await waitFor(() => expect(count("38")).toHaveValue(2));
    expect(screen.getByLabelText("Last pairs in")).toHaveTextContent("38, 39");
    // The page now knows 2, so the same number is sent against it.
    shop.setPairs.mockResolvedValueOnce(reply(200, { size: "38", pairs: 6, sizes: ["38", "39", "40"], sizes_low: ["39"], sizes_out: ["40"], stock: 8 }));
    change(count("38"), "6");
    fireEvent.blur(count("38"));
    await waitFor(() => expect(shop.setPairs).toHaveBeenLastCalledWith("tc-shoe", { size: "38", pairs: 6, seen_pairs: 2 }));
  });

  it("says it could not save and keeps the typed number when the shop cannot be reached", async () => {
    shop.setPairs.mockResolvedValueOnce(reply(0));
    open(stockTab);
    change(count("39"), "9");
    fireEvent.blur(count("39"));
    expect(await screen.findByText("We could not save. Please try again.")).toBeInTheDocument();
    expect(count("39")).toHaveValue(9);
    expect(screen.queryByText("Saved.")).toBeNull();
  });

  it("refuses a size added with no count, beside that size, and sends nothing", () => {
    open(["sizes", "name"]);
    fireEvent.change(screen.getByLabelText("Add to Sizes (EU)"), { target: { value: "41" } });
    fireEvent.keyDown(screen.getByLabelText("Add to Sizes (EU)"), { key: "Enter" });
    expect(count("41")).toHaveValue(null);
    fireEvent.click(screen.getByRole("button", { name: "Save changes" }));
    expect(screen.getByText("Enter the pairs in this size")).toHaveAttribute("role", "alert");
    expect(count("41")).toHaveAttribute("aria-invalid", "true");
    expect(shop.update).not.toHaveBeenCalled();
  });
});

describe("saving a shoe's page", () => {
  const basics = ["name", "price"];

  it("sends only the change, then draws what the shop answered and says Saved.", async () => {
    shop.update.mockResolvedValueOnce(reply(200, { ...shoe, price: 3490000 }));
    open(basics);
    type("Price", "3490000");
    fireEvent.click(screen.getByRole("button", { name: "Save changes" }));
    expect(await screen.findByText("Saved.")).toBeInTheDocument();
    expect(shop.update).toHaveBeenCalledWith("products", "tc-shoe", { price: 3490000 });
  });

  it("shows the name as the shop trimmed it, not as it was typed", async () => {
    shop.update.mockResolvedValueOnce(reply(200, shoe));
    open(basics);
    type("Name", "  Tc Shoe  ");
    fireEvent.click(screen.getByRole("button", { name: "Save changes" }));
    await screen.findByText("Saved.");
    expect(screen.getByLabelText("Name")).toHaveValue("Tc Shoe");
  });

  it("keeps the typed price and puts the shop's sentence beside it, announced, with no Saved.", async () => {
    shop.update.mockResolvedValueOnce(reply(422, { error: "invalid", message: "x", errors: { price: "Price cannot be below 0" } }));
    open(basics);
    type("Price", "-1");
    fireEvent.click(screen.getByRole("button", { name: "Save changes" }));
    expect(await screen.findByText("Price cannot be below 0")).toHaveAttribute("role", "alert");
    expect(screen.getByLabelText("Price")).toHaveValue(-1);
    expect(screen.getByLabelText("Price")).toHaveAttribute("aria-invalid", "true");
    expect(screen.queryByText("Saved.")).toBeNull();
  });

  it("says the session ended, or that the shop could not be reached, above the form", async () => {
    shop.update.mockResolvedValueOnce(reply(401));
    open(basics);
    type("Price", "1");
    fireEvent.click(screen.getByRole("button", { name: "Save changes" }));
    expect(await screen.findByRole("link", { name: "Sign in again." })).toHaveAttribute("href", "/admin/sign-in");
    shop.update.mockResolvedValueOnce(reply(502));
    fireEvent.click(screen.getByRole("button", { name: "Save changes" }));
    expect(await screen.findByText("We could not save. Please try again.")).toBeInTheDocument();
    expect(screen.getByLabelText("Price")).toHaveValue(1);
    expect(screen.getByRole("button", { name: "Save changes" })).toBeEnabled();
  });
});
