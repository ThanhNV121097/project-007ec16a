import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import Home from "../pages/Home";
import { SplitHeading } from "../motion";
import type { Product } from "../lib/shop";

// THE HOME PAGE OPENS ON THE SHOE THE DEALER MARKED "Opens the home page".
//
// The big line at the left is that shoe's own name, not a fixed sentence, so
// the dealer's choice in /admin/products shows in the heading as well as in
// the link and the picture. Only the shop's lists are stubbed.
vi.hoisted(() => vi.stubEnv("NODE_ENV", "test"));

const shop = vi.hoisted(() => ({ products: [] as unknown[] }));

vi.mock("../data", async (importOriginal) => {
  const real = await importOriginal<typeof import("../data")>();
  return {
    ...real,
    useCollection: (name: string) => ({
      items: name === "products" ? shop.products : [],
      total: 0, pages: 1, page: 1, hasMore: false, loading: false, error: null, loadMore: () => {},
    }),
  };
});

const shoe = (id: string, name: string, extra: Partial<Product> = {}): Product => ({
  id, name, line: "samba", category: "women", price: 3290000, code: "TC1", sizes: ["38", "39"], image: `/images/${id}.jpg`, status: "live", ...extra,
});

/** What the device says to the two media queries the motion code asks. */
function device(reducedMotion: boolean) {
  Object.defineProperty(window, "matchMedia", {
    configurable: true,
    writable: true,
    value: (query: string) => ({
      matches: reducedMotion && query.includes("prefers-reduced-motion"),
      media: query, addEventListener() {}, removeEventListener() {}, addListener() {}, removeListener() {},
    }),
  });
}

class Seen {
  observe() {}
  unobserve() {}
  disconnect() {}
}

const home = () => render(<MemoryRouter><Home /></MemoryRouter>);

beforeEach(() => {
  shop.products = [];
  device(true);
});
afterEach(() => {
  vi.unstubAllGlobals();
});

describe("the home page's lead shoe", () => {
  it("names the featured shoe in the heading, the link and the picture", () => {
    shop.products = [shoe("tc-feat-a", "Tc Feat A"), shoe("tc-feat-b", "Tc Feat B", { featured: true })];
    home();
    expect(screen.getByRole("heading", { level: 1 })).toHaveTextContent("Tc Feat B");
    expect(screen.getByTestId("home-lead")).toHaveAttribute("href", "/shop/tc-feat-b");
    expect(document.body).not.toHaveTextContent("Tc Feat A");
  });

  it("follows the choice when another shoe is featured, and falls back to the first live shoe", () => {
    shop.products = [shoe("tc-feat-a", "Tc Feat A", { featured: true }), shoe("tc-feat-b", "Tc Feat B")];
    const first = home();
    expect(screen.getByRole("heading", { level: 1 })).toHaveTextContent("Tc Feat A");
    first.unmount();

    shop.products = [shoe("tc-feat-b", "Tc Feat B"), shoe("tc-feat-a", "Tc Feat A")];
    home();
    expect(screen.getByRole("heading", { level: 1 })).toHaveTextContent("Tc Feat B");
    expect(screen.getByTestId("home-lead")).toHaveAttribute("href", "/shop/tc-feat-b");
  });
});

describe("a heading drawn from a record's words", () => {
  beforeEach(() => {
    device(false);
    vi.stubGlobal("IntersectionObserver", Seen);
  });

  it("rises word by word without claiming to be the Studio's text", () => {
    render(<SplitHeading text="Tc Feat B" as="h1" />);
    const heading = screen.getByRole("heading", { level: 1, name: "Tc Feat B" });
    expect(heading.querySelectorAll("[data-word]")).toHaveLength(3);
    expect(heading).not.toHaveAttribute("data-edit");
  });

  it("still draws a content key's words, marked for the Studio", () => {
    render(<SplitHeading k="pages.home.title" as="h1" />);
    const heading = screen.getByRole("heading", { level: 1 });
    expect(heading).toHaveAttribute("data-edit", "pages.home.title");
    expect(heading).toHaveAccessibleName("adidas shoes, delivered");
  });
});
