// @vitest-environment jsdom
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, render, screen, within } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";

// The sandbox sets NODE_ENV=production and React.act needs the development build.
vi.hoisted(() => {
  vi.stubEnv("NODE_ENV", "test");
});

import HeldOrders from "../components/HeldOrders";
import { heldOrders } from "../lib/track-an-order-by-number-and-phone";

// "Ordered on this phone": built from the orders this browser holds a token
// for, with no request to the shop.

const held = (n: number, day: number) => ({
  id: `doi-${n}`,
  number: `DOI-${n}`,
  placed_at: `2026-11-${day}T12:00:00+07:00`,
  token: `tc-token-${n}-abcdefghijklm`,
});

// Stored in an order that is not the order of their dates.
const SIX = [held(30003, 23), held(30006, 26), held(30001, 21), held(30005, 25), held(30002, 22), held(30004, 24)];

beforeEach(() => {
  localStorage.clear();
});

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
});

describe("the orders this browser holds", () => {
  it("are the five placed last, newest first by their placed time", () => {
    localStorage.setItem("doi:placed", JSON.stringify(SIX));
    expect(heldOrders().map((o) => o.number)).toEqual(["DOI-30006", "DOI-30005", "DOI-30004", "DOI-30003", "DOI-30002"]);
  });

  it("make no request to the shop", () => {
    const fetcher = vi.fn();
    vi.stubGlobal("fetch", fetcher);
    localStorage.setItem("doi:placed", JSON.stringify(SIX));
    heldOrders();
    expect(fetcher).not.toHaveBeenCalled();
  });

  it.each([
    ["nothing stored", null],
    ["an empty list", "[]"],
    ["something that is not a list", '{"not":"a list"}'],
    ["text that is not JSON", "{oops"],
  ])("are none for %s", (_name, stored) => {
    if (stored !== null) localStorage.setItem("doi:placed", stored);
    expect(heldOrders()).toEqual([]);
  });

  it("skip an entry that is not an order", () => {
    localStorage.setItem("doi:placed", JSON.stringify([null, 5, { id: "../x", number: "DOI-1", placed_at: "" }, held(30001, 21)]));
    expect(heldOrders().map((o) => o.number)).toEqual(["DOI-30001"]);
  });
});

describe("the list", () => {
  const draw = (orders = heldOrders()) =>
    render(
      <MemoryRouter>
        <HeldOrders orders={orders} />
      </MemoryRouter>,
    );

  it("has a row of number and date for each, linking to its own page", () => {
    localStorage.setItem("doi:placed", JSON.stringify(SIX));
    draw();
    expect(screen.getByRole("heading", { name: "Ordered on this phone" })).toBeInTheDocument();
    const rows = within(screen.getByTestId("track-orders")).getAllByRole("link");
    expect(rows.map((r) => r.getAttribute("href"))).toEqual(["/orders/doi-30006", "/orders/doi-30005", "/orders/doi-30004", "/orders/doi-30003", "/orders/doi-30002"]);
    expect(rows[0]).toHaveTextContent("DOI-30006");
    expect(rows[0]).toHaveTextContent("26/11/2026");
    expect(rows[4]).toHaveTextContent("22/11/2026");
    expect(screen.queryByText("DOI-30001")).toBeNull();
  });

  it("draws nothing at all when there are none", () => {
    const { container } = draw([]);
    expect(container).toBeEmptyDOMElement();
    expect(screen.queryByText("Ordered on this phone")).toBeNull();
  });
});
