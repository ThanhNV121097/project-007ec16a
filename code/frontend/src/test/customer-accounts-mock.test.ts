import { afterEach, describe, expect, it, vi } from "vitest";
import {
  addOrder,
  customerSignIn,
  customerSignOut,
  customerSignUp,
  getAccount,
  homeFor,
  listOrders,
  saveAccount,
} from "../lib/customer-accounts-sign-up-sign-in-my-orders";

/**
 * The story's API client against the shop's answers: the outcomes a screen
 * branches on, sorted from the status and the envelope code of each reply.
 * `fetch` is stubbed, so the real src/lib/api.ts runs and nothing touches a
 * network. The story's mock module is gone; this file keeps its path because
 * CI refuses a pull request that deletes a frontend file outside src/lib/mock/
 * and src/data/.
 */

type Route = (init?: RequestInit) => Response;

const reply = (body: unknown, status = 200): Response =>
  new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } });

const refusal = (status: number, error: string, errors?: Record<string, string>): Response =>
  reply(errors ? { error, message: "Refused.", errors } : { error, message: "Refused." }, status);

const lan = { name: "Lan Nguyen", email: "lan@example.com", phone: "0912 345 678", address: "", city: "" };
const row = { id: "doi-24818", number: "DOI-24818", placed_at: "2026-11-25T02:10:00Z", status: "placed", total: 3290000 };

/**
 * Stubs fetch with one answer per "METHOD path" (the path after /v1/, query
 * left out). /v1/auth/me answers 401 unless a route says otherwise; any other
 * request nobody listed fails the test.
 */
function stubShop(routes: Record<string, Route> = {}) {
  const mock = vi.fn(async (url: string, init?: RequestInit) => {
    const { pathname } = new URL(url, "http://localhost");
    const key = `${init?.method ?? "GET"} ${pathname.slice(pathname.indexOf("/v1/") + 4)}`;
    const route = routes[key];
    if (route) return route(init);
    if (key === "GET auth/me") return refusal(401, "unauthenticated");
    throw new Error(`unexpected request: ${key}`);
  });
  vi.stubGlobal("fetch", mock);
  return mock;
}

const keys = (mock: ReturnType<typeof stubShop>): string[] =>
  mock.mock.calls.map(([url, init]) => {
    const { pathname } = new URL(url, "http://localhost");
    return `${init?.method ?? "GET"} ${pathname.slice(pathname.indexOf("/v1/") + 4)}`;
  });

const sent = (init?: RequestInit): unknown => JSON.parse(String(init?.body));

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("signing in", () => {
  it("sends the typed email and password to the shared sign-in, then asks who is signed in", async () => {
    let body: unknown;
    const mock = stubShop({
      "POST auth/sign-in": (init) => {
        body = sent(init);
        return reply({ name: lan.name, email: lan.email, role: "customer" });
      },
      "GET auth/me": () => reply({ name: lan.name, email: lan.email, role: "customer" }),
    });
    const out = await customerSignIn("LAN@example.com", "lanpass123");
    expect(out).toEqual({ kind: "ok", value: { name: "Lan Nguyen", email: "lan@example.com", role: "customer" } });
    expect(body).toEqual({ email: "LAN@example.com", password: "lanpass123" });
    expect(keys(mock)).toEqual(["POST auth/sign-in", "GET auth/me"]);
  });

  it("reads a wrong password and an unknown email in the same way", async () => {
    stubShop({ "POST auth/sign-in": () => refusal(401, "invalid_credentials") });
    expect(await customerSignIn("lan@example.com", "wrong-password")).toEqual({ kind: "unauthenticated" });
    expect(await customerSignIn("nobody@example.com", "lanpass123")).toEqual({ kind: "unauthenticated" });
  });

  it("tells a lock and an unreachable shop from a refusal", async () => {
    stubShop({ "POST auth/sign-in": () => refusal(429, "rate_limited") });
    expect(await customerSignIn("lan@example.com", "lanpass123")).toEqual({ kind: "limited" });
    vi.stubGlobal("fetch", vi.fn(async () => {
      throw new TypeError("Failed to fetch");
    }));
    expect(await customerSignIn("lan@example.com", "lanpass123")).toEqual({ kind: "unreachable" });
  });

  it("never reads a gateway's answer as a wrong password", async () => {
    stubShop({ "POST auth/sign-in": () => refusal(401, "internal") });
    expect(await customerSignIn("lan@example.com", "lanpass123")).toEqual({ kind: "unreachable" });
    vi.stubGlobal("fetch", vi.fn(async () => new Response("<html>bad gateway</html>", { status: 502 })));
    expect(await customerSignIn("lan@example.com", "lanpass123")).toEqual({ kind: "unreachable" });
  });

  it("treats a 200 that is not an account as unreachable", async () => {
    stubShop({ "POST auth/sign-in": () => reply({ email: "lan@example.com" }) });
    expect(await customerSignIn("lan@example.com", "lanpass123")).toEqual({ kind: "unreachable" });
  });
});

describe("signing up", () => {
  const input = { name: "Lan Nguyen", email: "lan@example.com", phone: "0912 345 678", password: "lanpass123" };

  it("posts the four fields and signs the customer in", async () => {
    let body: unknown;
    const mock = stubShop({
      "POST account/sign-up": (init) => {
        body = sent(init);
        return reply({ name: "Lan Nguyen", email: "lan@example.com", role: "customer" }, 201);
      },
      "GET auth/me": () => reply({ name: "Lan Nguyen", email: "lan@example.com", role: "customer" }),
    });
    expect(await customerSignUp(input)).toEqual({ kind: "ok", value: { name: "Lan Nguyen", email: "lan@example.com", role: "customer" } });
    expect(body).toEqual(input);
    expect(keys(mock)).toEqual(["POST account/sign-up", "GET auth/me"]);
  });

  it("carries the sentence the shop sends for an email that is taken", async () => {
    stubShop({ "POST account/sign-up": () => refusal(409, "conflict", { email: "An account with this email already exists. Sign in instead." }) });
    expect(await customerSignUp(input)).toEqual({
      kind: "conflict",
      errors: { email: "An account with this email already exists. Sign in instead." },
    });
  });

  it("carries a sentence per field the shop refuses, and tells a lock from both", async () => {
    stubShop({ "POST account/sign-up": () => refusal(422, "invalid", { phone: "Check the phone number", password: "Use at least 8 characters" }) });
    expect(await customerSignUp(input)).toEqual({
      kind: "invalid",
      errors: { phone: "Check the phone number", password: "Use at least 8 characters" },
    });
    stubShop({ "POST account/sign-up": () => refusal(429, "rate_limited") });
    expect(await customerSignUp(input)).toEqual({ kind: "limited" });
  });
});

describe("my orders", () => {
  it("asks for ten at a time, a page at a time, and keeps the shape of a row", async () => {
    const mock = stubShop({ "GET account/orders": () => reply({ items: [row], total: 12, page: 2, limit: 10 }) });
    const out = await listOrders(2);
    expect(out).toEqual({ kind: "ok", value: { items: [row], total: 12, page: 2, limit: 10 } });
    expect(mock.mock.calls[0][0]).toMatch(/\/v1\/account\/orders\?page=2&limit=10$/);
    expect(Object.keys(row).sort()).toEqual(["id", "number", "placed_at", "status", "total"]);
  });

  it("tells no session from a team session", async () => {
    stubShop({ "GET account/orders": () => refusal(401, "unauthenticated") });
    expect(await listOrders(1)).toEqual({ kind: "unauthenticated" });
    stubShop({ "GET account/orders": () => refusal(403, "forbidden") });
    expect(await listOrders(1)).toEqual({ kind: "forbidden" });
  });

  it("is unreachable when the answer is a 500, after one more try", async () => {
    const mock = stubShop({ "GET account/orders": () => refusal(500, "internal") });
    expect(await listOrders(1)).toEqual({ kind: "unreachable" });
    expect(mock).toHaveBeenCalledTimes(2);
  });
});

describe("saved delivery details", () => {
  const details = { name: "Lan Nguyen", phone: "0912 345 678", address: "12 Example Street, Ward 1", city: "Da Nang" };

  it("reads the five fields of the account", async () => {
    stubShop({ "GET account": () => reply(lan) });
    expect(await getAccount()).toEqual({ kind: "ok", value: lan });
  });

  it("puts the four details, never the email, and asks who is signed in again", async () => {
    let body: unknown;
    const mock = stubShop({
      "PUT account": (init) => {
        body = sent(init);
        return reply({ ...lan, ...details });
      },
    });
    const out = await saveAccount(details);
    expect(out).toEqual({ kind: "ok", value: { ...lan, ...details } });
    expect(body).toEqual(details);
    expect(keys(mock)).toEqual(["PUT account", "GET auth/me"]);
  });

  it("carries the field the shop refuses", async () => {
    stubShop({ "PUT account": () => refusal(422, "invalid", { phone: "Check the phone number" }) });
    expect(await saveAccount({ ...details, phone: "0900" })).toEqual({ kind: "invalid", errors: { phone: "Check the phone number" } });
  });

  it("tells a team session from no session on both calls", async () => {
    stubShop({ "GET account": () => refusal(403, "forbidden"), "PUT account": () => refusal(401, "unauthenticated") });
    expect(await getAccount()).toEqual({ kind: "forbidden" });
    expect(await saveAccount(details)).toEqual({ kind: "unauthenticated" });
  });
});

describe("adding a guest order", () => {
  it("posts the number and the phone as typed and returns the row", async () => {
    let body: unknown;
    stubShop({
      "POST account/orders": (init) => {
        body = sent(init);
        return reply(row);
      },
    });
    expect(await addOrder("doi 24818", "+84912345678")).toEqual({ kind: "ok", value: row });
    expect(body).toEqual({ number: "doi 24818", phone: "+84912345678" });
  });

  it("is one not-found for a wrong pair, and a lock after too many", async () => {
    stubShop({ "POST account/orders": () => refusal(404, "not_found") });
    expect(await addOrder("24999", "0912345678")).toEqual({ kind: "notfound" });
    stubShop({ "POST account/orders": () => refusal(429, "rate_limited") });
    expect(await addOrder("24999", "0912345678")).toEqual({ kind: "limited" });
  });

  it("is not repeated after a 500: a POST is not idempotent", async () => {
    const mock = stubShop({ "POST account/orders": () => refusal(500, "internal") });
    expect(await addOrder("24818", "0912345678")).toEqual({ kind: "unreachable" });
    expect(mock).toHaveBeenCalledTimes(1);
  });
});

describe("signing out", () => {
  it("ends the session at the shop, then asks who is signed in", async () => {
    const mock = stubShop({ "POST auth/sign-out": () => new Response(null, { status: 204 }) });
    await customerSignOut();
    expect(keys(mock)).toEqual(["POST auth/sign-out", "GET auth/me"]);
  });
});

describe("where a signed-in person lands", () => {
  it("sends a customer to My account and the team to the dealer area", () => {
    expect(homeFor({ name: "Lan", email: "lan@example.com", role: "customer" })).toBe("/account");
    expect(homeFor({ name: "Minh", email: "minh@example.com", role: "staff" })).toBe("/admin");
    expect(homeFor({ name: "Dealer", email: "dealer@example.com", role: "admin" })).toBe("/admin");
  });
});
