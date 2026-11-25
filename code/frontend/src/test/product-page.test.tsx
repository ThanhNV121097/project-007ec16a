import { act, render, screen, waitFor } from "@testing-library/react";
import { useEffect } from "react";
import { MemoryRouter, Route, Routes, useNavigate, type NavigateFunction } from "react-router-dom";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import Product from "../pages/Product";
import type { Query } from "../data";
import type { Product as Shoe } from "../lib/shop";

// WHAT THE SHOE PAGE DRAWS, DECIDED BY WHAT THE RECORD REQUEST SAYS.
//
// `useRecord("products", id)` reports { item, loading, error }. Four answers,
// four views: a record in hand draws the shoe; waiting, and a shop that
// cannot answer, draw nothing; only an answered "not found" draws the
// missing view. The hook is stubbed so each answer is exactly the one under
// test; every other record and list the page asks for comes back empty.
type Answer = { item?: Shoe; loading: boolean; error: Error | null };

// React's `act`, which Testing Library renders through, exists only in React's
// development build, and React picks its build from NODE_ENV when it is first
// imported. A machine that exports NODE_ENV=production for its builds would
// run these tests against the production build and fail on `React.act`.
vi.hoisted(() => vi.stubEnv("NODE_ENV", "test"));

const hook = vi.hoisted(() => ({
  real: false,
  none: { item: undefined, loading: false, error: null } as Answer,
  answer: { item: undefined, loading: false, error: null } as Answer,
}));

// `hook.real` turns both stubs off for the tests at the end of the file, which
// run the real hook and the real API client against a stubbed `fetch`.
vi.mock("../data", async (importOriginal) => {
  const real = await importOriginal<typeof import("../data")>();
  return {
    ...real,
    useRecord: (name: string, id?: string) => (hook.real ? real.useRecord(name, id) : name === "products" ? hook.answer : hook.none),
    useCollection: (name: string, query?: Query) =>
      hook.real ? real.useCollection(name, query) : { items: [], total: 0, pages: 1, page: 1, hasMore: false, loading: false, error: null, loadMore: () => {} },
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

/**
 * A page that stays mounted while the address changes, as it does for a colour
 * dot or a related pair. The hook's answer is set first, then the address is
 * opened; a search suffix opens the same shoe again, which is how the page is
 * drawn again when the answer for its address arrives.
 */
function Navigator({ into }: { into: { current?: NavigateFunction } }) {
  const navigate = useNavigate();
  useEffect(() => {
    into.current = navigate;
  });
  return null;
}

function open(first: string) {
  const nav: { current?: NavigateFunction } = {};
  const view = render(
    <MemoryRouter initialEntries={[first]}>
      <Navigator into={nav} />
      <Routes><Route path="/shop/:id" element={<Product />} /></Routes>
    </MemoryRouter>,
  );
  const go = async (to: string, answer: Answer) => {
    hook.answer = answer;
    await act(async () => { nav.current?.(to); });
  };
  return { ...view, go };
}

beforeEach(() => {
  hook.answer = notFound;
  hook.real = false;
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
    const { container, go } = open(`/shop/${SHOE.id}`);
    expectShoe(SHOE.name);

    // The hook still holds the first shoe while the second one's request is in flight.
    await go(`/shop/${OTHER.id}`, { ...arrived(SHOE), loading: true });
    expectNothing(container);

    await go(`/shop/${OTHER.id}?answered`, arrived(OTHER));
    expectShoe(OTHER.name);
  });

  it("draws a live shoe after a 404, and the missing view only after the answer for a later unknown id", async () => {
    hook.answer = notFound;
    const { container, go } = open("/shop/not-a-shoe");
    expectMissing();

    await go(`/shop/${SHOE.id}`, arrived(SHOE));
    expectShoe(SHOE.name);

    await go("/shop/not-a-shoe", waiting);
    expectNothing(container);

    await go("/shop/not-a-shoe?answered", notFound);
    expectMissing();
  });

  it("draws a shoe after the shop could not be reached", async () => {
    hook.answer = failed;
    const { container, go } = open(`/shop/${SHOE.id}`);
    expectNothing(container);

    await go(`/shop/${OTHER.id}`, arrived(OTHER));
    expectShoe(OTHER.name);
  });
});

// THE SAME PAGE WITH THE REAL HOOK AND THE REAL API CLIENT.
//
// The tests above stub `useRecord`, so they prove the page's decision. These
// prove that the page and the hook agree: only `fetch` is stubbed, so the
// request in flight is a real one that the test holds, answers or fails. A
// server-side render cannot show the first paint (the data module's store has
// no server snapshot), so a MutationObserver watches the page from before its
// first commit and remembers every node that was ever put on it: the missing
// text must never be among them while the shop has not said "not found".
const json = (body: unknown, status = 200) =>
  new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } });
const NOT_FOUND = { error: "not_found", message: "Not found." };

type Answerer = () => Response | Promise<Response>;

/** A request the test answers when it chooses. */
function held() {
  let release: (r: Response) => void = () => {};
  const answer = new Promise<Response>((resolve) => {
    release = resolve;
  });
  return { answer: () => answer, release: (r: Response) => act(async () => release(r)) };
}

/** The shop at fetch: each shoe answers as given (not found when not given); lists are empty, other records not found. */
function shop(shoes: Record<string, Answerer>) {
  const mock = vi.fn(async (input: string) => {
    const path = new URL(input, "http://localhost").pathname;
    const own = /^\/api\/v1\/products\/([^/]+)$/.exec(path)?.[1];
    if (own !== undefined) return shoes[own]?.() ?? json(NOT_FOUND, 404);
    if (path === "/api/v1/products" || path === "/api/v1/reviews") return json({ items: [], total: 0, page: 1, limit: 12 });
    return json(NOT_FOUND, 404);
  });
  vi.stubGlobal("fetch", mock);
  return (id: string) => mock.mock.calls.filter((c) => c[0] === `/api/v1/products/${id}`).length;
}

/** The page at `first`, observed from before its first commit. */
function watched(first: string) {
  const root = document.body.appendChild(document.createElement("div"));
  const added: string[] = [];
  const note = (records: MutationRecord[]) => records.forEach((r) => r.addedNodes.forEach((n) => added.push(n.textContent ?? "")));
  const observer = new MutationObserver(note);
  observer.observe(root, { childList: true, subtree: true });
  const nav: { current?: NavigateFunction } = {};
  render(
    <MemoryRouter initialEntries={[first]}>
      <Navigator into={nav} />
      <Routes><Route path="/shop/:id" element={<Product />} /></Routes>
    </MemoryRouter>,
    { container: root },
  );
  return {
    root,
    go: (to: string) => act(async () => { nav.current?.(to); }),
    everMissing: () => {
      note(observer.takeRecords());
      return added.some((text) => text.includes(MISSING));
    },
  };
}

const settle = () => act(async () => { await new Promise((resolve) => setTimeout(resolve, 20)); });

describe("the real hook, with the shop's answer held", () => {
  beforeEach(() => {
    hook.real = true;
  });
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("draws nothing from the first paint until the shoe arrives, then the shoe whole", async () => {
    const gate = held();
    const asked = shop({ "held-shoe": gate.answer });
    const page = watched("/shop/held-shoe");
    expectNothing(page.root);
    await settle();
    expect(asked("held-shoe")).toBe(1);
    expectNothing(page.root);

    await gate.release(json({ ...SHOE, id: "held-shoe" }));
    await screen.findByTestId("product-name");
    expectShoe(SHOE.name);
    expect(page.everMissing()).toBe(false);
  });

  it("draws the missing view only after the shop answers not found", async () => {
    const gate = held();
    shop({ "gone-shoe": gate.answer });
    const page = watched("/shop/gone-shoe");
    await settle();
    expectNothing(page.root);
    expect(page.everMissing()).toBe(false);

    await gate.release(json(NOT_FOUND, 404));
    await screen.findByText(MISSING);
    expectMissing();
  });

  const failures: [string, Answerer, number][] = [
    ["a 500 from the API", () => json({ error: "internal", message: "Something went wrong." }, 500), 2],
    ["a gateway's 502 page", () => new Response("<html>Bad gateway</html>", { status: 502 }), 2],
    ["a 404 that is not the API's own", () => new Response("<html>404 Not Found</html>", { status: 404 }), 1],
    ["no connection", () => Promise.reject(new TypeError("Failed to fetch")), 2],
  ];
  it.each(failures)("never says the pair is gone when the shop gives %s", async (label, answer, tries) => {
    const id = `failing-${label.replace(/\W+/g, "-")}`;
    const asked = shop({ [id]: answer });
    const page = watched(`/shop/${id}`);
    await waitFor(() => expect(asked(id)).toBe(tries));
    await settle();
    expect(asked(id)).toBe(tries);
    expectNothing(page.root);
    expect(page.everMissing()).toBe(false);
  });

  it("draws neither the previous shoe nor the missing view while a colour dot's shoe is awaited", async () => {
    const next = held();
    shop({ "dot-first": () => json({ ...SHOE, id: "dot-first" }), "dot-second": next.answer });
    const page = watched("/shop/dot-first");
    await screen.findByTestId("product-name");
    expectShoe(SHOE.name);

    await page.go("/shop/dot-second");
    await settle();
    expectNothing(page.root);

    await next.release(json({ ...OTHER, id: "dot-second" }));
    await screen.findByText(OTHER.name);
    expectShoe(OTHER.name);
    expect(page.everMissing()).toBe(false);
  });

  it("keeps the shoe of the address when an earlier address answers last", async () => {
    const early = held();
    shop({ "late-a": early.answer, "late-b": () => json({ ...OTHER, id: "late-b" }) });
    const page = watched("/shop/late-a");
    await settle();
    await page.go("/shop/late-b");
    await screen.findByTestId("product-name");

    await early.release(json({ ...SHOE, id: "late-a" }));
    await settle();
    expectShoe(OTHER.name);
    expect(page.everMissing()).toBe(false);
  });

  it("carries nothing from a 404 to the next shoe, nor from a shoe to the next unknown address", async () => {
    const slow = held();
    shop({ "seq-live": () => json({ ...SHOE, id: "seq-live" }), "seq-slow-gone": slow.answer });
    const page = watched("/shop/seq-gone");
    await screen.findByText(MISSING);
    expectMissing();

    await page.go("/shop/seq-live");
    await screen.findByTestId("product-name");
    expectShoe(SHOE.name);

    await page.go("/shop/seq-slow-gone");
    await settle();
    expectNothing(page.root);

    await slow.release(json(NOT_FOUND, 404));
    await screen.findByText(MISSING);
    expectMissing();
  });
});
