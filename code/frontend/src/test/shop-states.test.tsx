// @vitest-environment jsdom
import { fireEvent, render, screen } from "@testing-library/react";
import { MemoryRouter, useLocation } from "react-router-dom";
import { describe, expect, it, vi } from "vitest";

// React's act() needs the development build; the sandbox may export production.
vi.hoisted(() => {
  vi.stubEnv("NODE_ENV", "test");
});

import FilterRail from "../components/FilterRail";
import ShopFilterToggle from "../components/ShopFilterToggle";
import ShopResults, { type ShopResultsProps } from "../components/ShopResults";
import ShopSizes from "../components/ShopSizes";
import { useTableQuery } from "../lib/table";
import type { Product } from "../lib/shop";

function shoe(id: string): Product {
  return { id, name: `Pair ${id}`, line: "samba", category: "women", price: 2500000 };
}

function results(over: Partial<ShopResultsProps> = {}) {
  const props: ShopResultsProps = {
    items: [], total: null, hasMore: false, loading: false, error: null, lines: new Map(),
    onMore: vi.fn(), onRetry: vi.fn(), onClear: vi.fn(), ...over,
  };
  render(<MemoryRouter><ShopResults {...props} /></MemoryRouter>);
  return props;
}

const pair = [shoe("a"), shoe("b")];

describe("the results area", () => {
  it("says the shoes are on their way, and never that nothing matches", () => {
    results({ loading: true });
    expect(screen.getByRole("status").textContent).toBe("Loading the shoes…");
    expect(screen.queryByText("No pair matches that.")).toBeNull();
    expect(screen.queryByRole("link")).toBeNull();
  });

  it("says the shoes could not be loaded, and asks again on Try again", () => {
    const p = results({ error: new Error("500") });
    expect(screen.getByRole("alert").textContent).toBe("We could not load the shoes.");
    expect(screen.queryByText("No pair matches that.")).toBeNull();
    expect(screen.queryByText("Loading the shoes…")).toBeNull();
    expect(document.body.textContent).not.toMatch(/500|Error/);
    fireEvent.click(screen.getByRole("button", { name: "Try again" }));
    expect(p.onRetry).toHaveBeenCalledTimes(1);
  });

  it("shows the empty state only for a list that came back empty", () => {
    const p = results({ total: 0 });
    expect(screen.getByRole("heading", { name: "No pair matches that." })).toBeTruthy();
    expect(screen.getByText("Try another size or fewer filters — or ask us, and we will tell you when it comes in.")).toBeTruthy();
    const clear = screen.getByRole("button", { name: "Clear all" });
    expect(clear.querySelector("svg")).not.toBeNull();
    fireEvent.click(clear);
    expect(p.onClear).toHaveBeenCalledTimes(1);
    expect(screen.queryByRole("status")).toBeNull();
    expect(screen.queryByRole("alert")).toBeNull();
  });

  it("lists the shoes with Show more under them", () => {
    const p = results({ items: pair, total: 30, hasMore: true });
    expect(screen.getAllByRole("link")).toHaveLength(2);
    fireEvent.click(screen.getByRole("button", { name: "Show more" }));
    expect(p.onMore).toHaveBeenCalledTimes(1);
  });

  it("disables Show more and reads Loading… while its page is on its way, keeping the shoes", () => {
    const p = results({ items: pair, total: 30, hasMore: true, loading: true });
    const button = screen.getByRole("button", { name: "Loading…" }) as HTMLButtonElement;
    expect(button.disabled).toBe(true);
    expect(screen.getAllByRole("link")).toHaveLength(2);
    fireEvent.click(button);
    expect(p.onMore).not.toHaveBeenCalled();
    expect(screen.queryByRole("status")).toBeNull();
  });

  it("puts the message and Try again where Show more was when its page fails, and keeps the shoes", () => {
    const p = results({ items: pair, total: 30, hasMore: true, error: new Error("500") });
    expect(screen.getAllByRole("link")).toHaveLength(2);
    expect(screen.getByRole("alert").textContent).toBe("We could not load the shoes.");
    expect(screen.queryByRole("button", { name: "Show more" })).toBeNull();
    expect(screen.getAllByRole("button", { name: "Try again" })).toHaveLength(1);
    fireEvent.click(screen.getByRole("button", { name: "Try again" }));
    expect(p.onRetry).toHaveBeenCalledTimes(1);
  });
});

describe("the phone's Filter button", () => {
  function toggle(count: number, open = false) {
    const onToggle = vi.fn();
    render(<ShopFilterToggle open={open} count={count} controls="shop-filters" onToggle={onToggle} />);
    return { button: screen.getByRole("button"), onToggle };
  }

  it("reads Filter with nothing set and Filter (n) with a count", () => {
    expect(toggle(0).button.textContent).toBe("Filter");
  });

  it("puts a space before the bracket", () => {
    expect(toggle(3).button.textContent).toBe("Filter (3)");
  });

  it("reports whether the rail is open and asks to switch it", () => {
    const closed = toggle(0);
    expect(closed.button.getAttribute("aria-expanded")).toBe("false");
    expect(closed.button.getAttribute("aria-controls")).toBe("shop-filters");
    fireEvent.click(closed.button);
    expect(closed.onToggle).toHaveBeenCalledTimes(1);
  });

  it("carries aria-expanded true when open", () => {
    expect(toggle(1, true).button.getAttribute("aria-expanded")).toBe("true");
  });
});

describe("a group with a single option", () => {
  it("draws no Size (EU) heading over one pill, or none", () => {
    const { container, rerender } = render(<ShopSizes sizes={["40"]} pressed={[]} onToggle={vi.fn()} />);
    expect(container.textContent).toBe("");
    rerender(<ShopSizes sizes={[]} pressed={[]} onToggle={vi.fn()} />);
    expect(container.textContent).toBe("");
  });

  it("draws the group with two sizes, each pill pressed or not", () => {
    render(<ShopSizes sizes={["38", "40"]} pressed={["38"]} onToggle={vi.fn()} />);
    expect(screen.getByText("Size (EU)")).toBeTruthy();
    expect(screen.getByTestId("filter-size-38").getAttribute("aria-pressed")).toBe("true");
    expect(screen.getByTestId("filter-size-40").getAttribute("aria-pressed")).toBe("false");
  });
});

/** The rail as the shop draws it: the page's own query, handed down. */
function Rail() {
  const t = useTableQuery({ sizeIsFilter: true });
  const address = useLocation().search;
  return (
    <>
      <output data-testid="address">{address}</output>
      <FilterRail table={t} groups={[{ key: "category", label: "For", options: [["women", "Women"], ["men", "Men"]] }]} />
    </>
  );
}

function rail(address: string) {
  render(<MemoryRouter initialEntries={[address]}><Rail /></MemoryRouter>);
  return () => screen.getByTestId("address").textContent;
}

describe("Clear all under the rail", () => {
  it("removes the filters and the size and keeps the order", () => {
    const address = rail("/shop?category=women&size=38&sort=price");
    fireEvent.click(screen.getByRole("button", { name: "Clear all" }));
    expect(address()).toBe("?sort=price");
  });

  it("is shown for a lone size, and removes it", () => {
    const address = rail("/shop?size=38");
    fireEvent.click(screen.getByRole("button", { name: "Clear all" }));
    expect(address()).toBe("");
  });

  it("is not shown with nothing set, nor with only an order", () => {
    rail("/shop");
    expect(screen.queryByRole("button", { name: "Clear all" })).toBeNull();
  });

  it("is not shown with only a sort", () => {
    rail("/shop?sort=price");
    expect(screen.queryByRole("button", { name: "Clear all" })).toBeNull();
  });

  it("appears once a box is ticked", () => {
    rail("/shop");
    fireEvent.click(screen.getByRole("checkbox", { name: "Women" }));
    expect(screen.getByRole("button", { name: "Clear all" })).toBeTruthy();
  });
});
