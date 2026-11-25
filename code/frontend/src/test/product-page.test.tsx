import { act, render, screen } from "@testing-library/react";
import { createMemoryRouter, MemoryRouter, Route, RouterProvider, Routes } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import Product from "../pages/Product";
import type { Product as Shoe } from "../lib/shop";

// WHAT THE SHOE PAGE DRAWS, DECIDED BY WHAT THE RECORD REQUEST SAYS.
//
// `useRecord("products", id)` reports { item, loading, error }. Four answers,
// four views: a record in hand draws the shoe; waiting, and a shop that
// cannot answer, draw nothing; only an answered "not found" draws the
// missing view. The hook is stubbed so each answer is exactly the one under
// test; every other record and list the page asks for comes back empty.
type Answer = { item?: Shoe; loading: boolean; error: Error | null };

const hook = vi.hoisted(() => ({
  none: { item: undefined, loading: false, error: null } as Answer,
  answer: { item: undefined, loading: false, error: null } as Answer,
}));

vi.mock("../data", async (importOriginal) => {
  const real = await importOriginal<typeof import("../data")>();
  return {
    ...real,
    useRecord: (name: string) => (name === "products" ? hook.answer : hook.none),
    useCollection: () => ({ items: [], total: 0, pages: 1, page: 1, hasMore: false, loading: false, error: null, loadMore: () => {} }),
  };
});

const MISSING = "That pair is not in the shop any more.";
const SHOE: Shoe = {
  id: "samba-og-w-crystal-white",
  name: "Samba OG Crystal White",
  line: "samba",
  category: "women",
  price: 3090000,
  code: "IH9055",
  sizes: ["36", "38", "40"],
  image: "/images/samba.jpg",
};
const OTHER: Shoe = { ...SHOE, id: "samba-og-w-wonder-quartz", name: "Samba OG Wonder Quartz" };

const arrived = (item: Shoe): Answer => ({ item, loading: false, error: null });
const waiting: Answer = { item: undefined, loading: true, error: null };
const failed: Answer = { item: undefined, loading: false, error: new Error("The shop could not be reached") };
const notFound: Answer = { item: undefined, loading: false, error: null };

function at(id: string) {
  return render(
    <MemoryRouter initialEntries={[`/shop/${id}`]}>
      <Routes><Route path="/shop/:id" element={<Product />} /></Routes>
    </MemoryRouter>,
  );
}

/** The page holds the shoe: its name as the heading, its price, its sizes and the button. */
function expectShoe(name: string) {
  expect(screen.getByTestId("product-name")).toHaveTextContent(name);
  expect(screen.getByTestId("add-to-bag")).toBeInTheDocument();
  expect(screen.getByTestId("size-38")).toBeInTheDocument();
  expect(screen.getByText(/3\.090\.000/)).toBeInTheDocument();
  expect(screen.queryByText(MISSING)).toBeNull();
}

/** Nothing between the header and the footer: no heading, no link, no button. */
function expectNothing(container: HTMLElement) {
  expect(container).toBeEmptyDOMElement();
  expect(screen.queryByText(MISSING)).toBeNull();
  expect(screen.queryByRole("link", { name: /all shoes/i })).toBeNull();
  expect(screen.queryByTestId("product-name")).toBeNull();
}

/** The missing view: the h1 and the one "All shoes" pill to /shop, and no part of a shoe. */
function expectMissing() {
  expect(screen.getByRole("heading", { level: 1 })).toHaveTextContent(MISSING);
  const pills = screen.getAllByRole("link", { name: /all shoes/i });
  expect(pills).toHaveLength(1);
  expect(pills[0]).toHaveAttribute("href", "/shop");
  expect(screen.queryByTestId("product-name")).toBeNull();
  expect(screen.queryByTestId("add-to-bag")).toBeNull();
  expect(screen.queryByTestId("size-38")).toBeNull();
}

beforeEach(() => {
  hook.answer = notFound;
});

describe("a record for this address is in hand", () => {
  it("draws the shoe, and not the missing view", () => {
    hook.answer = arrived(SHOE);
    at(SHOE.id);
    expectShoe(SHOE.name);
    expect(screen.getByRole("link", { name: /all shoes/i })).toHaveAttribute("href", "/shop");
  });

  it("draws the shoe whatever the other flags say", () => {
    hook.answer = { ...arrived(SHOE), loading: true };
    const first = at(SHOE.id);
    expectShoe(SHOE.name);
    first.unmount();

    hook.answer = { ...arrived(SHOE), error: new Error("a later refresh failed") };
    at(SHOE.id);
    expectShoe(SHOE.name);
  });
});

describe("the shop has not answered", () => {
  it("draws nothing while the request is in flight", () => {
    hook.answer = waiting;
    const { container } = at("not-a-shoe");
    expectNothing(container);
  });
});

describe("the shop cannot answer", () => {
  it("draws nothing, and never says the pair is gone", () => {
    hook.answer = failed;
    const { container } = at(SHOE.id);
    expectNothing(container);
  });
});

describe("the shop answered not found", () => {
  it("draws the missing view and its pill to /shop", () => {
    hook.answer = notFound;
    at("not-a-shoe");
    expectMissing();
  });
});

describe("one address does not leak into the next", () => {
  it("never shows the previous shoe under a new address, nor the missing view before its answer", () => {
    hook.answer = arrived(OTHER);
    const stale = at(SHOE.id);
    expectNothing(stale.container);
    stale.unmount();

    hook.answer = { ...arrived(OTHER), loading: true };
    const loading = at(SHOE.id);
    expectNothing(loading.container);
  });

  it("follows a colour dot to a shoe whose answer is slow, then draws it", async () => {
    hook.answer = arrived(SHOE);
    const router = createMemoryRouter([{ path: "/shop/:id", element: <Product /> }], { initialEntries: [`/shop/${SHOE.id}`] });
    const { container } = render(<RouterProvider router={router} />);
    expectShoe(SHOE.name);

    // The hook still holds the first shoe while the second one's request is in flight.
    hook.answer = { ...arrived(SHOE), loading: true };
    await act(async () => { await router.navigate(`/shop/${OTHER.id}`); });
    expectNothing(container);

    hook.answer = arrived(OTHER);
    await act(async () => { await router.navigate(`/shop/${OTHER.id}?again`); });
    expectShoe(OTHER.name);
  });

  it("draws a live shoe after a 404, and the missing view only after the answer for a later unknown id", async () => {
    hook.answer = notFound;
    const router = createMemoryRouter([{ path: "/shop/:id", element: <Product /> }], { initialEntries: ["/shop/not-a-shoe"] });
    const { container } = render(<RouterProvider router={router} />);
    expectMissing();

    hook.answer = arrived(SHOE);
    await act(async () => { await router.navigate(`/shop/${SHOE.id}`); });
    expectShoe(SHOE.name);

    hook.answer = waiting;
    await act(async () => { await router.navigate("/shop/not-a-shoe"); });
    expectNothing(container);

    hook.answer = notFound;
    await act(async () => { await router.navigate("/shop/not-a-shoe?again"); });
    expectMissing();
  });

  it("draws a shoe after the shop could not be reached", async () => {
    hook.answer = failed;
    const router = createMemoryRouter([{ path: "/shop/:id", element: <Product /> }], { initialEntries: [`/shop/${SHOE.id}`] });
    const { container } = render(<RouterProvider router={router} />);
    expectNothing(container);

    hook.answer = arrived(OTHER);
    await act(async () => { await router.navigate(`/shop/${OTHER.id}`); });
    expectShoe(OTHER.name);
  });
});
