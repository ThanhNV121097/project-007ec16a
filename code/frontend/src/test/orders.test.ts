import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { buildRequest, detailProblems, keyKeeper, payAddress, placeOrder, rememberOrder, sizeOfVariant, storedOrders, type Placed } from "../lib/orders";

// The browser's side of POST /v1/orders: what it sends, what each answer means
// to the page, and what this browser keeps of an order. fetch is stubbed.

const details = { name: " Lan Nguyen ", phone: "0912 345 678", address: "12 Example Street, Ward 1", city: "Da Nang", note: "" };
const line = { id: "samba-og-w-crystal-white", name: "Samba OG", price: 3000000, variant: "EU 36 2/3", qty: 2 };
const order = { id: "doi-24818", number: "DOI-24818", placed_at: "2026-11-25T10:00:00+07:00", status: "placed", paid: false, total: 6180000 };
const placed = (url = "") => ({ order, token: "q3J0n8Zk2LmPa7Vx9TrBwE", payment_url: url });
const send = vi.fn();

function answer(status: number, body: unknown) {
  send.mockResolvedValueOnce({ status, json: () => Promise.resolve(body) });
}

beforeEach(() => {
  localStorage.clear();
  send.mockReset();
  vi.stubGlobal("fetch", send);
});

afterEach(() => {
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
});

describe("the order request", () => {
  it("names the pairs and the details and carries no price, fee or total", () => {
    const r = buildRequest([line], details);
    expect(r.items).toEqual([{ product: "samba-og-w-crystal-white", size: "36 2/3", qty: 2 }]);
    expect(r).toMatchObject({ delivery: "standard", payment: "vnpay", customer: "Lan Nguyen", phone: "0912 345 678", note: "" });
    expect(JSON.stringify(r)).not.toMatch(/price|fee|total/i);
  });

  it("reads a size out of a bag line's variant", () => {
    expect(sizeOfVariant("EU 38")).toBe("38");
    expect(sizeOfVariant("EU 43 1/3")).toBe("43 1/3");
    expect(sizeOfVariant(undefined)).toBe("");
  });

  it("is one POST to /v1/orders with the key as a header, once, with the cookie", async () => {
    answer(201, placed());
    const request = buildRequest([line], details);
    await placeOrder(request, "tc-key-0001");
    expect(send).toHaveBeenCalledTimes(1);
    const [url, init] = send.mock.calls[0] as [string, RequestInit];
    expect(url).toMatch(/\/v1\/orders$/);
    expect(init.method).toBe("POST");
    expect(init.credentials).toBe("include");
    expect(init.headers).toMatchObject({ "Idempotency-Key": "tc-key-0001", "Content-Type": "application/json" });
    expect(JSON.parse(init.body as string)).toEqual(request);
  });
});

describe("the details the shop would refuse", () => {
  it("wants a name, an address and a city", () => {
    expect(detailProblems(details)).toEqual([]);
    expect(detailProblems({ ...details, name: "" })).toEqual(["missing"]);
    expect(detailProblems({ ...details, address: "   " })).toEqual(["missing"]);
    expect(detailProblems({ ...details, city: "   " })).toEqual(["missing"]);
  });

  it("counts the digits of a phone and nothing else", () => {
    expect(detailProblems({ ...details, phone: "0912 34" })).toEqual(["phone"]);
    expect(detailProblems({ ...details, phone: "09123456" })).toEqual(["phone"]);
    expect(detailProblems({ ...details, phone: "(091) 234-56" })).toEqual(["phone"]);
    expect(detailProblems({ ...details, phone: "091234567" })).toEqual([]);
    expect(detailProblems({ ...details, phone: "+84 912 345 678" })).toEqual([]);
  });

  it("names both problems, the missing details first", () => {
    expect(detailProblems({ ...details, name: "", phone: "12" })).toEqual(["missing", "phone"]);
  });
});

describe("the idempotency key", () => {
  it("is one key for one bag with one set of details, and a new one when either changes", () => {
    const keyFor = keyKeeper();
    const key = keyFor(buildRequest([line], details));
    expect(key).toMatch(/^[A-Za-z0-9._-]{8,100}$/);
    expect(keyFor(buildRequest([line], details))).toBe(key);
    const other = keyFor(buildRequest([line], { ...details, phone: "0912 345 679" }));
    expect(other).not.toBe(key);
    expect(keyFor(buildRequest([{ ...line, qty: 3 }], { ...details, phone: "0912 345 679" }))).not.toBe(other);
  });
});

describe("what an answer means", () => {
  it("201 is the placed order with its token, and an empty payment address stays empty", async () => {
    answer(201, placed());
    expect(await placeOrder(buildRequest([line], details), "tc-key-0001")).toEqual({ ok: true, placed: placed() });
  });

  it("a replay (200) is the same order again", async () => {
    answer(200, placed());
    expect((await placeOrder(buildRequest([line], details), "tc-key-0001")).ok).toBe(true);
  });

  it("carries the payment address the shop gives", async () => {
    answer(201, placed("https://pay.example.test/x"));
=======
    send.mockResolvedValue({ status: 201, body: placed("https://pay.example.test/x") });
>>>>>>> refs/heads/main
    const r = await placeOrder(buildRequest([line], details), "tc-key-0002");
    expect(r.ok && r.placed.payment_url).toBe("https://pay.example.test/x");
  });

  it("409 stock_conflict names every pair, in the order of the bag", async () => {
    answer(409, { error: "stock_conflict", message: "x", errors: { "items.10": "B · EU 38 — sold out in your size.", "items.2": "A · EU 40 — only 2 left in your size." } });
=======
    send.mockResolvedValue({
      status: 409,
      body: { error: "stock_conflict", message: "x", errors: { "items.10": "B · EU 38 — sold out in your size.", "items.2": "A · EU 40 — only 2 left in your size." } },
    });
>>>>>>> refs/heads/main
    const r = await placeOrder(buildRequest([line], details), "tc-key-0003");
    expect(r).toEqual({ ok: false, kind: "refused", problems: ["A · EU 40 — only 2 left in your size.", "B · EU 38 — sold out in your size."] });
  });

  it("422 is invalid, 503 payment_unavailable is unavailable", async () => {
    answer(422, { error: "invalid", message: "x", errors: { address: "Enter the street address." } });
    expect(await placeOrder(buildRequest([line], details), "tc-key-0004")).toEqual({ ok: false, kind: "invalid", problems: ["Enter the street address."] });
    answer(503, { error: "payment_unavailable", message: "x" });
=======
    send.mockResolvedValueOnce({ status: 422, body: { error: "invalid", message: "x", errors: { address: "Enter an address." } } });
    expect(await placeOrder(buildRequest([line], details), "tc-key-0004")).toEqual({ ok: false, kind: "invalid", problems: ["Enter an address."] });
    send.mockResolvedValueOnce({ status: 503, body: { error: "payment_unavailable", message: "x" } });
>>>>>>> refs/heads/main
    expect(await placeOrder(buildRequest([line], details), "tc-key-0004")).toEqual({ ok: false, kind: "unavailable", problems: [] });
  });

  it("500, another 409, an answer with no order and a network failure are all a failed send", async () => {
    answer(500, { error: "internal", message: "x" });
    answer(409, { error: "conflict", message: "x" });
    answer(201, {});
=======
    send.mockResolvedValueOnce({ status: 500, body: { error: "internal", message: "x" } });
    send.mockResolvedValueOnce({ status: 409, body: { error: "conflict", message: "x" } });
    send.mockResolvedValueOnce({ status: 201, body: {} });
>>>>>>> refs/heads/main
    send.mockRejectedValueOnce(new TypeError("Failed to fetch"));
    for (let i = 0; i < 4; i++) {
      expect(await placeOrder(buildRequest([line], details), "tc-key-0005")).toEqual({ ok: false, kind: "failed", problems: [] });
    }
  });

  it("an answer that is not JSON is a failed send", async () => {
    send.mockResolvedValueOnce({ status: 502, json: () => Promise.reject(new SyntaxError("Unexpected token <")) });
    expect(await placeOrder(buildRequest([line], details), "tc-key-0005")).toEqual({ ok: false, kind: "failed", problems: [] });
  });

  it("an offline browser says so and sends nothing", async () => {
    vi.spyOn(window.navigator, "onLine", "get").mockReturnValue(false);
    expect(await placeOrder(buildRequest([line], details), "tc-key-0006")).toEqual({ ok: false, kind: "offline", problems: [] });
    expect(send).not.toHaveBeenCalled();
  });
});

describe("where the browser goes to pay", () => {
  it("follows an https address and nothing else", () => {
    expect(payAddress("https://pay.example.test/x")).toBe("https://pay.example.test/x");
    for (const bad of ["", "http://pay.example.test/x", "javascript:alert(1)", "//pay.example.test/x", "/orders/doi-1"]) {
      expect(payAddress(bad)).toBeNull();
    }
  });
});

describe("the orders this browser has placed", () => {
  const one = (id: string): Placed => ({ order: { ...order, id, number: id.toUpperCase() }, token: `token-of-${id}-xxxxxxxx`, payment_url: "" });

  it("keeps the id, number, placed time and token and nothing else, under a key of its own", () => {
    localStorage.setItem("doi:orders", "[]");
    rememberOrder(one("doi-30001"));
    const saved = JSON.parse(localStorage.getItem("doi:placed") ?? "null") as unknown;
    expect(saved).toEqual([{ id: "doi-30001", number: "DOI-30001", placed_at: order.placed_at, token: "token-of-doi-30001-xxxxxxxx" }]);
    expect(localStorage.getItem("doi:orders")).toBe("[]");
  });

  it("still holds the order for this page when the browser refuses storage", () => {
    vi.spyOn(Storage.prototype, "setItem").mockImplementation(() => {
      throw new Error("storage is full");
    });
    expect(() => rememberOrder(one("doi-30002"))).not.toThrow();
    expect(storedOrders().find((o) => o.id === "doi-30002")?.token).toBe("token-of-doi-30002-xxxxxxxx");
  });
});
