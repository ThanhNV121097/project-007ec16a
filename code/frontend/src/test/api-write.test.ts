import { afterEach, describe, expect, it, vi } from "vitest";
import { api } from "../lib/api";

const answer = (status: number, body?: unknown) =>
  ({ ok: status >= 200 && status < 300, status, json: async () => body }) as unknown as Response;

afterEach(() => vi.unstubAllGlobals());

describe("the api client's PATCH and upload", () => {
  it("PATCHes JSON with the session cookie and does not retry", async () => {
    const fetchMock = vi.fn(async () => answer(200, { size: "38" }));
    vi.stubGlobal("fetch", fetchMock);
    await expect(api.patch("products/x/sizes", { size: "38" })).resolves.toEqual({ size: "38" });
    const [url, init] = fetchMock.mock.calls[0] as unknown as [string, RequestInit];
    expect(url).toMatch(/\/v1\/products\/x\/sizes$/);
    expect(init.method).toBe("PATCH");
    expect(init.credentials).toBe("include");
    expect((init.headers as Record<string, string>)["Content-Type"]).toBe("application/json");
    expect(init.body).toBe('{"size":"38"}');
  });

  it("does not try a failed PATCH a second time", async () => {
    const fetchMock = vi.fn(async () => answer(500, { error: "internal", message: "No." }));
    vi.stubGlobal("fetch", fetchMock);
    await expect(api.patch("products/x/sizes", {})).rejects.toMatchObject({ status: 500, code: "internal" });
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });

  it("POSTs a form as it is, with no JSON content type", async () => {
    const fetchMock = vi.fn(async () => answer(201, { url: "/api/v1/uploads/a.png" }));
    vi.stubGlobal("fetch", fetchMock);
    const form = new FormData();
    form.append("file", new File(["x"], "a.png", { type: "image/png" }));
    await expect(api.upload("uploads", form)).resolves.toEqual({ url: "/api/v1/uploads/a.png" });
    const [, init] = fetchMock.mock.calls[0] as unknown as [string, RequestInit];
    expect(init.method).toBe("POST");
    expect(init.body).toBe(form);
    expect((init.headers as Record<string, string>)["Content-Type"]).toBeUndefined();
  });
});
