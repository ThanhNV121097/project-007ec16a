// @vitest-environment jsdom
import { vi } from "vitest";

// The sandbox sets NODE_ENV=production and React.act needs the development build.
vi.hoisted(() => {
  vi.stubEnv("NODE_ENV", "test");
});

// The shop answers each list by its name; the layout and the menu's own count are not what these cases are about.
vi.mock("../lib/api", async (original) => {
  const actual = await original<typeof import("../lib/api")>();
  return { ...actual, api: { ...actual.api, get: vi.fn() } };
});
vi.mock("../pages/admin-nav", async (original) => ({
  ...(await original<typeof import("../pages/admin-nav")>()),
  useAdminNav: () => [],
}));
vi.mock("../components/DashboardLayout", () => ({
  default: ({ title, children }: { title: string; children: import("react").ReactNode }) => (
    <div>
      <h1>{title}</h1>
      <main>{children}</main>
    </div>
  ),
}));

import { cleanup, render, screen, waitFor, within } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { api } from "../lib/api";
import AdminProducts from "../pages/AdminProducts";
import AdminReviews from "../pages/AdminReviews";

const read = vi.mocked(api.get);

// What the shop holds, by list. Each case sets the records it is about, under its own ids.
let held: Record<string, unknown[]> = {};

beforeEach(() => {
  held = {};
  read.mockImplementation((path: string) => {
    const items = held[path] ?? [];
    return Promise.resolve({ items, total: items.length, page: 1, limit: 20 });
  });
});
afterEach(cleanup);

const open = (page: import("react").ReactElement) => render(<MemoryRouter>{page}</MemoryRouter>);

describe("a list that points at another record shows that record's name", () => {
  it("names each shoe's line and category on the Shoes list, never their ids", async () => {
    held.products = [{ id: "tc-ref-a", name: "Tc Ref Shoe A", line: "tc-ref-line", category: "tc-ref-kind", price: 3290000, status: "live", stock: 6 }];
    held.lines = [{ id: "tc-ref-line", name: "Tc Ref Line" }];
    held.categories = [{ id: "tc-ref-kind", name: "Tc Ref Kind" }];
    open(<AdminProducts />);
    const row = (await screen.findByText("Tc Ref Shoe A")).closest("tr") as HTMLElement;
    await waitFor(() => expect(within(row).getByText("Tc Ref Line")).toBeInTheDocument());
    expect(within(row).getByText("Tc Ref Kind")).toBeInTheDocument();
    expect(within(row).queryByText("tc-ref-line")).toBeNull();
    expect(within(row).queryByText("tc-ref-kind")).toBeNull();
  });

  it("names each review's shoe on the Reviews list, never its id", async () => {
    held.products = [{ id: "tc-ref-b", name: "Tc Ref Shoe B", line: "samba", category: "women", price: 3290000, status: "draft" }];
    held.reviews = [{ id: "r-tcref001", product: "tc-ref-b", name: "Tc Buyer", rating: 5, date: "2026-11-01", status: "live" }];
    open(<AdminReviews />);
    const row = (await screen.findByText("Tc Buyer")).closest("tr") as HTMLElement;
    await waitFor(() => expect(within(row).getByText("Tc Ref Shoe B")).toBeInTheDocument());
    expect(within(row).queryByText("tc-ref-b")).toBeNull();
  });
});
