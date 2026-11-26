import { afterEach, describe, expect, it, vi } from "vitest";
import { ApiError, api } from "../lib/api";

// The one API client: base URL, same-origin cookies, the error envelope read
// into an ApiError, a 15-second timeout and one retry of a failed GET. fetch
// is stubbed; nothing here touches a network.

function reply(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } });
}

type Fetch = (url: string, init?: RequestInit) => Promise<Response>;

function stubFetch(impl: Fetch) {
  const mock = vi.fn(impl);
  vi.stubGlobal("fetch", mock);
  return mock;
}

afterEach(() => {
  vi.unstubAllGlobals();
  vi.unstubAllEnvs();
  vi.useRealTimers();
});

describe("where a request goes", () => {
  it("uses VITE_API_URL when it is set and /api when it is not", async () => {
    const mock = stubFetch(async () => reply({ items: [] }));
    await api.get("products");
    expect(mock.mock.calls[0][0]).toBe("/api/v1/products");
    vi.stubEnv("VITE_API_URL", "https://api.example.test");
    await api.get("products", { page: 1, limit: 12 });
    expect(mock.mock.calls[1][0]).toBe("https://api.example.test/v1/products?page=1&limit=12");
  });

  it("uses an empty VITE_API_URL as given", async () => {
    vi.stubEnv("VITE_API_URL", "");
    const mock = stubFetch(async () => reply({}));
    await api.get("shop/shop");
    expect(mock.mock.calls[0][0]).toBe("/v1/shop/shop");
  });

  it("sends the cookie, leaves undefined out and keeps a comma as written", async () => {
    const mock = stubFetch(async () => reply({}));
    await api.get("products", { category: "women,kids", q: undefined, featured: true, price: 3290000 });
    expect(mock.mock.calls[0][0]).toBe("/api/v1/products?category=women,kids&featured=true&price=3290000");
    expect(mock.mock.calls[0][1]?.credentials).toBe("include");
  });

  it("escapes a value that holds a slash or a space", async () => {
    const mock = stubFetch(async () => reply({}));
    await api.get("products", { sizes_low: "36 2/3" });
    expect(mock.mock.calls[0][0]).toBe("/api/v1/products?sizes_low=36%202%2F3");
  });
});

describe("a failed answer", () => {
  it("becomes an ApiError carrying the envelope, and a 4xx is not tried again", async () => {
    const envelope = { error: "bad_request", message: "Some parameters are not valid.", errors: { page: "page must be a whole number from 1." } };
    const mock = stubFetch(async () => reply(envelope, 400));
    const err = await api.get("products", { page: 0 }).catch((e: unknown) => e);
    expect(err).toBeInstanceOf(ApiError);
    expect(err).toMatchObject({ status: 400, code: "bad_request", message: "Some parameters are not valid.", errors: envelope.errors });
    expect(mock).toHaveBeenCalledTimes(1);
  });

  it("never retries a 404", async () => {
    const mock = stubFetch(async () => reply({ error: "not_found", message: "Not found." }, 404));
    const err = await api.get("products/nope").catch((e: unknown) => e);
    expect(err).toMatchObject({ status: 404, code: "not_found" });
    expect(mock).toHaveBeenCalledTimes(1);
  });

  it("tries a 5xx once more before it fails", async () => {
    const mock = stubFetch(async () => reply({ error: "internal", message: "Something went wrong." }, 500));
    const err = await api.get("products").catch((e: unknown) => e);
    expect(err).toMatchObject({ status: 500, code: "internal" });
    expect(mock).toHaveBeenCalledTimes(2);
  });

  it("succeeds when the second try does", async () => {
    let n = 0;
    const mock = stubFetch(async () => (++n === 1 ? reply({ error: "internal", message: "x" }, 503) : reply({ ok: true })));
    await expect(api.get("products")).resolves.toEqual({ ok: true });
    expect(mock).toHaveBeenCalledTimes(2);
  });

  it("treats no connection as status 0 and tries once more", async () => {
    const mock = stubFetch(async () => {
      throw new TypeError("Failed to fetch");
    });
    const err = await api.get("products").catch((e: unknown) => e);
    expect(err).toBeInstanceOf(ApiError);
    expect(err).toMatchObject({ status: 0, code: "network" });
    expect(mock).toHaveBeenCalledTimes(2);
  });

  it("reads a body that is not the envelope as a generic internal error", async () => {
    stubFetch(async () => new Response("<html>bad gateway</html>", { status: 502 }));
    const err = await api.get("products").catch((e: unknown) => e);
    expect(err).toMatchObject({ status: 502, code: "internal" });
    expect((err as ApiError).message).not.toContain("html");
  });
});

describe("post", () => {
  it("sends the body as JSON with the cookie, and never tries again", async () => {
    const mock = stubFetch(async () => reply({ error: "internal", message: "x" }, 500));
    const err = await api.post("auth/sign-in", { email: "a@b.example", password: "pw" }).catch((e: unknown) => e);
    expect(err).toMatchObject({ status: 500, code: "internal" });
    expect(mock).toHaveBeenCalledTimes(1);
    const [url, init] = mock.mock.calls[0];
    expect(url).toBe("/api/v1/auth/sign-in");
    expect(init).toMatchObject({ method: "POST", credentials: "include", body: JSON.stringify({ email: "a@b.example", password: "pw" }) });
    expect((init?.headers as Record<string, string>)["Content-Type"]).toBe("application/json");
  });

  it("sends no content type without a body and takes a 204 for success", async () => {
    const mock = stubFetch(async () => new Response(null, { status: 204 }));
    await expect(api.post("auth/sign-out")).resolves.toBeUndefined();
    const init = mock.mock.calls[0][1];
    expect(init?.body).toBeUndefined();
    expect((init?.headers as Record<string, string>)["Content-Type"]).toBeUndefined();
  });

  it("carries the envelope of a refusal, and no connection is status 0", async () => {
    stubFetch(async () => reply({ error: "rate_limited", message: "Too many tries." }, 429));
    await expect(api.post("auth/sign-in", {})).rejects.toMatchObject({ status: 429, code: "rate_limited" });
    const down = stubFetch(async () => {
      throw new TypeError("Failed to fetch");
    });
    await expect(api.post("auth/sign-in", {})).rejects.toMatchObject({ status: 0, code: "network" });
    expect(down).toHaveBeenCalledTimes(1);
  });
});

describe("time and sharing", () => {
  it("aborts a request unanswered after 15 seconds and counts it as a failure", async () => {
    vi.useFakeTimers();
    const signals: AbortSignal[] = [];
    const mock = stubFetch((_url, init) => {
      if (init?.signal) signals.push(init.signal);
      return new Promise<Response>(() => {});
    });
    const outcome = api.get("products").catch((e: unknown) => e);
    await vi.advanceTimersByTimeAsync(14_999);
    expect(signals[0].aborted).toBe(false);
    await vi.advanceTimersByTimeAsync(1);
    expect(signals[0].aborted).toBe(true);
    await vi.advanceTimersByTimeAsync(15_000);
    const err = await outcome;
    expect(err).toMatchObject({ status: 0, code: "network" });
    expect(mock).toHaveBeenCalledTimes(2);
  });

  it("lets identical simultaneous reads share one request", async () => {
    const mock = stubFetch(async () => reply({ id: "samba" }));
    const [a, b] = await Promise.all([api.get("lines/samba"), api.get("lines/samba")]);
    expect(a).toEqual({ id: "samba" });
    expect(b).toEqual({ id: "samba" });
    expect(mock).toHaveBeenCalledTimes(1);
    await api.get("lines/samba");
    expect(mock).toHaveBeenCalledTimes(2);
  });
});
