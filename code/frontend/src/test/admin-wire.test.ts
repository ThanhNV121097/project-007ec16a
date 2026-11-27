import { beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("../lib/api", async (original) => {
  const real = await original<typeof import("../lib/api")>();
  return {
    ...real,
    api: { get: vi.fn(), post: vi.fn(), put: vi.fn(), patch: vi.fn(), delete: vi.fn(), upload: vi.fn() },
  };
});

import { api, ApiError } from "../lib/api";
import transport from "../lib/manage-shoes-lines-categories-reviews-and-orders-wire";

const shop = vi.mocked(api);

beforeEach(() => vi.resetAllMocks());

describe("the writes' transport over api.ts", () => {
  it("sends a new record to its collection and says 201 with what the shop answered", async () => {
    shop.post.mockResolvedValue({ id: "samba-og" });
    const wire = await transport.create("products", { name: "Samba OG" });
    expect(shop.post).toHaveBeenCalledWith("products", { name: "Samba OG" });
    expect(wire).toEqual({ status: 201, body: { id: "samba-og" } });
  });

  it("addresses a change and a removal by the id, as one path segment", async () => {
    shop.put.mockResolvedValue({ id: "a b" });
    shop.delete.mockResolvedValue(undefined);
    expect((await transport.update("lines", "a b", { name: "X" })).status).toBe(200);
    expect(shop.put).toHaveBeenCalledWith("lines/a%20b", { name: "X" });
    expect((await transport.remove("reviews", "r-1")).status).toBe(204);
    expect(shop.delete).toHaveBeenCalledWith("reviews/r-1");
  });

  it("keeps the shop's refusal: its status and its envelope with the fields at fault", async () => {
    shop.post.mockRejectedValue(new ApiError(422, "invalid", "Check the form.", { price: "Price cannot be below 0" }));
    const wire = await transport.create("products", { price: -1 });
    expect(wire).toEqual({
      status: 422,
      body: { error: "invalid", message: "Check the form.", errors: { price: "Price cannot be below 0" } },
    });
  });

  it("says 0 when no answer came, and for anything that is not the shop's answer", async () => {
    shop.get.mockRejectedValueOnce(new ApiError(0, "network", "The server cannot be reached."));
    expect((await transport.read("products", "x")).status).toBe(0);
    shop.get.mockRejectedValueOnce(new TypeError("boom"));
    expect(await transport.read("products", "x")).toEqual({ status: 0 });
  });

  it("changes one size's count with PATCH on the shoe's sizes", async () => {
    shop.patch.mockResolvedValue({ size: "38", pairs: 6 });
    const wire = await transport.setPairs("tc-shoe", { size: "38", pairs: 6, seen_pairs: 4 });
    expect(shop.patch).toHaveBeenCalledWith("products/tc-shoe/sizes", { size: "38", pairs: 6, seen_pairs: 4 });
    expect(wire.status).toBe(200);
  });

  it("sends a picture as a form with one part, file", async () => {
    shop.upload.mockResolvedValue({ url: "/api/v1/uploads/abc.jpg" });
    const file = new File(["x"], "side.jpg", { type: "image/jpeg" });
    const wire = await transport.upload(file);
    const [path, form] = shop.upload.mock.calls[0] as [string, FormData];
    expect(path).toBe("uploads");
    expect([...form.keys()]).toEqual(["file"]);
    expect(wire).toEqual({ status: 201, body: { url: "/api/v1/uploads/abc.jpg" } });
  });
});
