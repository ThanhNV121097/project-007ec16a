// @vitest-environment jsdom
import { vi } from "vitest";

// The sandbox sets NODE_ENV=production and React.act needs the development build.
vi.hoisted(() => {
  vi.stubEnv("NODE_ENV", "test");
});

// GET /v1/admin/overview is the page's one request; each case says what it answers.
// The real ApiError stays, so a refusal is the error the client throws. The layout
// and the menu's own count are not what these cases are about.
vi.mock("../lib/api", async (original) => {
  const actual = await original<typeof import("../lib/api")>();
  return { ...actual, api: { ...actual.api, get: vi.fn() } };
});
vi.mock("../pages/admin-nav", () => ({ useAdminNav: () => [] }));
vi.mock("../components/DashboardLayout", () => ({
  default: ({ title, children }: { title: string; children: import("react").ReactNode }) => (
    <div>
      <h1>{title}</h1>
      <main>{children}</main>
    </div>
  ),
}));

import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { ApiError, api } from "../lib/api";
import { dayOfMonth, type Overview } from "../lib/overview";
import AdminOverview from "../pages/AdminOverview";

const read = vi.mocked(api.get);

/** The SRS's answer at its fixed clock (25/11/2026): 13 orders, 47.660.000 ₫, two shoes running low, five lines, five latest orders. */
const sampleOverview: Overview = {
  month: { sales: 47660000, orders: 13 },
  waiting_orders: 2,
  pairs_on_shelf: 73,
  shoes: 3,
  running_low: [
    { id: "gazelle-indoor-solar-gold", name: "Gazelle Indoor Solar Gold", category: "men", image: "/images/ref/gazelle-indoor-solar-gold.jpg", stock: 5, sizes_low: ["40", "41"] },
    { id: "samba-og-kids-cloud-white", name: "Samba OG Kids Cloud White", category: "kids", image: "/images/ref/samba-og-kids-cloud-white.jpg", stock: 9, sizes_low: ["36"] },
  ],
  sales_by_day: [
    { date: "2026-11-12", sales: 2790000 }, { date: "2026-11-13", sales: 3590000 },
    { date: "2026-11-14", sales: 3090000 }, { date: "2026-11-15", sales: 3790000 },
    { date: "2026-11-16", sales: 0 }, { date: "2026-11-17", sales: 2290000 },
    { date: "2026-11-18", sales: 5090000 }, { date: "2026-11-19", sales: 3290000 },
    { date: "2026-11-20", sales: 0 }, { date: "2026-11-21", sales: 5580000 },
    { date: "2026-11-22", sales: 3590000 }, { date: "2026-11-23", sales: 5380000 },
    { date: "2026-11-24", sales: 5890000 }, { date: "2026-11-25", sales: 3290000 },
  ],
  pairs_by_line: [
    { line: "samba", name: "Samba", pairs: 8 },
    { line: "gazelle", name: "Gazelle", pairs: 2 },
    { line: "megaride", name: "Megaride", pairs: 2 },
    { line: "handball-spezial", name: "Handball Spezial", pairs: 1 },
    { line: "adizero", name: "Adizero", pairs: 1 },
  ],
  latest_orders: [
    { id: "doi-24816", number: "DOI-24816", customer: "Trang", product: "handball-spezial-w-earth-strata", product_name: "Handball Spezial Earth Strata", size: "37 1/3", status: "placed", placed_at: "2026-11-25T08:40:00+07:00", total: 3290000 },
    { id: "doi-24815", number: "DOI-24815", customer: "Long", product: "megaride-bones-halo-green", product_name: "Megaride Bones Halo Green", size: "43 1/3", status: "confirmed", placed_at: "2026-11-24T10:12:00+07:00", total: 5890000 },
    { id: "doi-24814", number: "DOI-24814", customer: "Hương", product: "samba-og-kids-cloud-white", product_name: "Samba OG Kids Cloud White", size: "37 1/3", status: "packed", placed_at: "2026-11-23T19:30:00+07:00", total: 2290000 },
    { id: "doi-24817", number: "DOI-24817", customer: "Hoàng Nam", product: "samba-og-w-crystal-white", product_name: "Samba OG Crystal White", size: "39 1/3", status: "shipped", placed_at: "2026-11-23T16:05:00+07:00", total: 3090000 },
    { id: "doi-24813", number: "DOI-24813", customer: "Tuấn", product: "gazelle-indoor-solar-gold", product_name: "Gazelle Indoor Solar Gold", size: "43 1/3", status: "shipped", placed_at: "2026-11-22T11:02:00+07:00", total: 3590000 },
  ],
};

/** A shop with nothing sold and nothing running low: every list empty, every day 0. */
const emptyOverview: Overview = {
  month: { sales: 0, orders: 0 },
  waiting_orders: 0,
  pairs_on_shelf: 0,
  shoes: 0,
  running_low: [],
  sales_by_day: sampleOverview.sales_by_day.map((d) => ({ date: d.date, sales: 0 })),
  pairs_by_line: [],
  latest_orders: [],
};

/** White space as one space, so a no-break space before the dong sign reads as a space. */
const flat = (text: string | null | undefined) => (text ?? "").replace(/\s+/g, " ").trim();

function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((res) => {
    resolve = res;
  });
  return { promise, resolve };
}

function open() {
  return render(
    <MemoryRouter>
      <AdminOverview />
    </MemoryRouter>,
  );
}

/** The card whose title reads `name` (the figures' labels are descriptions, not titles). */
function panel(container: HTMLElement, name: string): HTMLElement {
  const title = Array.from(container.querySelectorAll('[data-slot="card-title"]')).find((el) => el.textContent === name);
  if (!title) throw new Error(`no panel called ${name}`);
  return title.closest('[data-slot="card"]') as HTMLElement;
}

const figure = (label: string) =>
  screen.getByText(label, { selector: '[data-slot="card-description"]' }).closest("a") as HTMLAnchorElement;

const five: Overview = {
  ...sampleOverview,
  running_low: [
    ...sampleOverview.running_low,
    { id: "a", name: "A", category: "men", image: "/a.jpg", stock: 11, sizes_low: ["38"] },
    { id: "b", name: "B", category: "men", image: "/b.jpg", stock: 12, sizes_low: ["42"] },
    { id: "c", name: "C", category: "men", image: "/c.jpg", stock: 14, sizes_low: ["43 1/3", "44"] },
  ],
};

beforeEach(() => {
  read.mockReset();
});
afterEach(cleanup);

describe("the figures", () => {
  it("shows the title and four figures read from the answer, each opening its list", async () => {
    read.mockResolvedValue(sampleOverview);
    open();
    await screen.findByText("Sales this month");
    expect(read).toHaveBeenCalledTimes(1);
    expect(read).toHaveBeenCalledWith("admin/overview");
    expect(screen.getByRole("heading", { name: "Overview" })).toBeInTheDocument();
    const labels = ["Sales this month", "To pack", "Pairs in stock", "Running low"];
    expect(labels.map((l) => figure(l).getAttribute("href"))).toEqual([
      "/admin/orders", "/admin/orders?status=confirmed", "/admin/products", "/admin/products",
    ]);
    const text = labels.map((l) => flat(figure(l).textContent));
    expect(text[0]).toContain("47.660.000 ₫");
    expect(text[0]).toContain("13 orders");
    expect(text[1]).toContain("2");
    expect(text[1]).toContain("placed or confirmed");
    expect(text[2]).toContain("73");
    expect(text[2]).toContain("across 3 shoes");
    expect(text[3]).toContain("2");
    expect(text[3]).toContain("last pairs left");
    expect(document.body.textContent).not.toMatch(/NaN|undefined|null|\[object/);
  });

  it("writes the Today sentence from the same answer as the figures", async () => {
    read.mockResolvedValue(five);
    open();
    const sentence = await screen.findByText(/orders are waiting to be packed/);
    expect(flat(sentence.textContent)).toBe(
      "2 orders are waiting to be packed and 5 shoes are down to their last pairs. This month so far: 13 orders, 47.660.000 ₫.",
    );
    expect(flat(figure("Running low").textContent)).toContain("5");
  });

  it("reads the empty shop as zeros, short columns and no rows", async () => {
    read.mockResolvedValue(emptyOverview);
    const { container } = open();
    await screen.findByText("Nothing is running low.");
    expect(flat(figure("Sales this month").textContent)).toContain("0 ₫");
    expect(flat(figure("Sales this month").textContent)).toContain("0 orders");
    expect(panel(container, "Latest orders").querySelector("li")).toBeNull();
    expect(panel(container, "Pairs sold by line").querySelector("li")).toBeNull();
    const columns = panel(container, "Sales, last 14 days").querySelectorAll("[title]");
    expect(columns).toHaveLength(14);
    columns.forEach((c) => expect(flat(c.getAttribute("title"))).toMatch(/^\d+: 0 ₫$/));
    expect(container.querySelector('a[href^="/admin/products/"]')).toBeNull();
  });
});

describe("the charts", () => {
  it("draws 14 columns in the answer's order, labelled with the shop's day of the month", async () => {
    read.mockResolvedValue(sampleOverview);
    const { container } = open();
    await screen.findByText("Sales this month");
    const chart = panel(container, "Sales, last 14 days").querySelector("figure") as HTMLElement;
    const columns = Array.from(chart.querySelectorAll("[title]"));
    expect(columns).toHaveLength(14);
    expect(flat(columns[0].getAttribute("title"))).toBe("12: 2.790.000 ₫");
    expect(flat(columns[2].getAttribute("title"))).toBe("14: 3.090.000 ₫");
    expect(flat(columns[4].getAttribute("title"))).toBe("16: 0 ₫");
    expect(flat(columns[12].getAttribute("title"))).toBe("24: 5.890.000 ₫");
    const labels = Array.from((chart.lastElementChild as HTMLElement).children).map((c) => c.textContent);
    expect(labels).toEqual(["12", "13", "14", "15", "16", "17", "18", "19", "20", "21", "22", "23", "24", "25"]);
    const bar = (i: number) => columns[i].querySelector("span") as HTMLElement;
    expect(bar(12).style.height).toBe("100%");
    expect(bar(12)).toHaveClass("bg-accent");
    expect(bar(4)).toHaveClass("bg-wash");
    expect(bar(4).style.height).toBe("2%");
  });

  it("takes the day from the date text, never through a Date", () => {
    expect(dayOfMonth("2026-11-12")).toBe("12");
    expect(dayOfMonth("2026-12-01")).toBe("1");
    expect(dayOfMonth("2026-11-30")).toBe("30");
  });

  it("draws one row per line in the answer's order, its bar in proportion to the largest", async () => {
    read.mockResolvedValue({ ...sampleOverview, pairs_by_line: [...sampleOverview.pairs_by_line, { line: "terrex", name: "Terrex", pairs: 0 }] });
    const { container } = open();
    await screen.findByText("Sales this month");
    const rows = Array.from(panel(container, "Pairs sold by line").querySelectorAll("li"));
    expect(rows.map((r) => r.textContent)).toEqual(["Samba8", "Gazelle2", "Megaride2", "Handball Spezial1", "Adizero1", "Terrex0"]);
    expect(rows.map((r) => (r.querySelector("[style]") as HTMLElement).style.width)).toEqual(["100%", "25%", "25%", "12.5%", "12.5%", "0%"]);
  });
});

describe("the rows", () => {
  it("lists the latest orders as links, a cancelled one reading Cancelled", async () => {
    read.mockResolvedValue({
      ...sampleOverview,
      latest_orders: sampleOverview.latest_orders.map((o, i) => (i === 2 ? { ...o, status: "cancelled" } : o)),
    });
    const { container } = open();
    await screen.findByText("Latest orders");
    const rows = Array.from(panel(container, "Latest orders").querySelectorAll("a"));
    expect(rows.map((a) => a.getAttribute("href"))).toEqual([
      "/admin/orders/doi-24816", "/admin/orders/doi-24815", "/admin/orders/doi-24814", "/admin/orders/doi-24817", "/admin/orders/doi-24813",
    ]);
    const [left, right] = Array.from(rows[0].children).map((c) => flat(c.textContent));
    expect(left).toBe("DOI-24816 · Trang · Handball Spezial Earth Strata · EU 37 1/3");
    expect(right).toMatch(/^Placed · \d{2}\/\d{2}\/2026, \d{1,2}:\d{2} · 3\.290\.000 ₫$/);
    expect(flat(rows[2].lastElementChild?.textContent)).toMatch(/^Cancelled · .* · 2\.290\.000 ₫$/);
    expect(panel(container, "Latest orders").textContent).not.toContain("cancelled");
  });

  it("lists the shoes running low with their sizes and pairs, each a link to the shoe", async () => {
    read.mockResolvedValue(sampleOverview);
    const { container } = open();
    await screen.findByText("Last pairs in 40, 41");
    const rows = Array.from(panel(container, "Running low").querySelectorAll("a"));
    expect(rows.map((a) => a.getAttribute("href"))).toEqual([
      "/admin/products/gazelle-indoor-solar-gold", "/admin/products/samba-og-kids-cloud-white",
    ]);
    expect(flat(rows[0].textContent)).toContain("Gazelle Indoor Solar Gold");
    expect(flat(rows[0].textContent)).toMatch(/Last pairs in 40, 41\s*5$/);
    expect(flat(rows[1].textContent)).toMatch(/Last pairs in 36\s*9$/);
    expect(screen.queryByText("Nothing is running low.")).toBeNull();
  });
});

describe("waiting and failing", () => {
  it("shows the title and a polite line while the answer is awaited, and no figure", async () => {
    const held = deferred<Overview>();
    read.mockReturnValueOnce(held.promise);
    const { container } = open();
    expect(screen.getByRole("heading", { name: "Overview" })).toBeInTheDocument();
    expect(screen.getByRole("status")).toHaveTextContent("Loading the overview…");
    expect(screen.queryByText("Sales this month")).toBeNull();
    expect(screen.queryByText("Nothing is running low.")).toBeNull();
    expect(container.querySelector("figure")).toBeNull();
    expect(container.querySelector('[data-slot="card"]')).toBeNull();
    await act(async () => {
      held.resolve(sampleOverview);
    });
    expect(screen.getByText("Sales this month")).toBeInTheDocument();
    expect(screen.queryByRole("status")).toBeNull();
  });

  it.each([
    ["a server error", new ApiError(500, "internal", "Something went wrong. Please try again.")],
    ["no connection", new ApiError(0, "network", "The server cannot be reached.")],
    ["a visitor", new ApiError(401, "unauthenticated", "Sign in to continue.")],
    ["a customer", new ApiError(403, "forbidden", "You do not have access to this.")],
  ])("says it could not load, with Try again, and no figure — %s", async (_cause, failure) => {
    read.mockRejectedValue(failure);
    const { container } = open();
    expect(await screen.findByRole("alert")).toHaveTextContent("We could not load the overview.");
    expect(screen.getByRole("button", { name: "Try again" })).toBeInTheDocument();
    expect(screen.queryByText("Sales this month")).toBeNull();
    expect(container.querySelector('[data-slot="card"]')).toBeNull();
    expect(screen.queryByText("0")).toBeNull();
  });

  it("takes an answer of the wrong shape for a failed read", async () => {
    read.mockResolvedValue({ month: { sales: 1, orders: 1 } } as unknown as Overview);
    open();
    expect(await screen.findByRole("alert")).toHaveTextContent("We could not load the overview.");
    expect(screen.queryByText("Sales this month")).toBeNull();
  });

  it("shows the loading line again on Try again, asks again, and the message returns if it fails again", async () => {
    read.mockRejectedValue(new Error("down"));
    open();
    await screen.findByRole("alert");
    expect(read).toHaveBeenCalledTimes(1);

    const held = deferred<Overview>();
    read.mockReturnValueOnce(held.promise);
    fireEvent.click(screen.getByRole("button", { name: "Try again" }));
    expect(screen.getByRole("status")).toHaveTextContent("Loading the overview…");
    expect(screen.queryByRole("alert")).toBeNull();
    expect(read).toHaveBeenCalledTimes(2);
    await act(async () => {
      held.resolve(sampleOverview);
    });
    expect(screen.getByText("Sales this month")).toBeInTheDocument();
    expect(screen.queryByRole("alert")).toBeNull();
    expect(screen.queryByRole("status")).toBeNull();
  });

  it("brings the message back when the second try fails too", async () => {
    read.mockRejectedValue(new Error("down"));
    open();
    await screen.findByRole("alert");
    fireEvent.click(screen.getByRole("button", { name: "Try again" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("We could not load the overview.");
    expect(read).toHaveBeenCalledTimes(2);
  });
});
