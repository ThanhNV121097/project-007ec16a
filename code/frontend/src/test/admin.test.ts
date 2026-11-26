import { beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("../lib/manage-shoes-lines-categories-reviews-and-orders-wire", () => ({
  default: { create: vi.fn(), update: vi.fn(), remove: vi.fn(), read: vi.fn(), upload: vi.fn(), setPairs: vi.fn() },
}));

import {
  blank, columnsFromFields, fieldsOf, labelOf, missing, titleOf, type FieldSpec,
} from "../components/admin/fields";
import { bodyOf, EMPTY_COUNT, stockErrors, valuesOf, WHOLE_COUNT } from "../components/admin/payload";
import { deleteRow, newId, saveCount, saveRow, uploadFile } from "../components/admin/store";
import { schema, setCollection, type Row } from "../data";
import transport from "../lib/manage-shoes-lines-categories-reviews-and-orders-wire";

const shop = vi.mocked(transport);
const reply = (status: number, body?: unknown) => ({ status, body });
const TAKEN = "Pairs were just taken by an order. Check the number and save again.";

beforeEach(() => {
  Object.values(shop).forEach((fn) => fn.mockReset());
});

// THE MANAGEMENT SCREENS' PIECES OF LOGIC.
//
// Everything else in src/components/admin is markup, which the free walk and
// a reviewer can see. These are what a screen DERIVES — the columns, the
// form, what is refused, what is sent — and they are invisible: a wrong one
// ships a CMS that looks finished and edits the wrong record.
describe("the fields a management screen derives", () => {
  it("names a field a person can read", () => {
    expect(labelOf("created_at")).toBe("Created at");
    expect(labelOf("fullName")).toBe("Full name");
  });

  it("draws for a collection the schema does not describe", () => {
    expect(fieldsOf("nothing-here")).toEqual([]);
    expect(columnsFromFields([])).toEqual([]);
  });

  it("keeps long text out of the table and numbers to the right", () => {
    const fields: FieldSpec[] = [
      { key: "name", label: "Name", type: "text" },
      { key: "story", label: "Story", type: "longtext" },
      { key: "price", label: "Price", type: "price" },
    ];
    const cols = columnsFromFields(fields);
    expect(cols.map((c) => c.key)).toEqual(["name", "price"]);
    expect(cols.find((c) => c.key === "price")?.align).toBe("right");
  });

  it("refuses a required field that is empty, and only that", () => {
    const fields: FieldSpec[] = [
      { key: "name", label: "Name", type: "text", required: true },
      { key: "note", label: "Note", type: "text" },
      { key: "live", label: "Live", type: "bool", required: true },
    ];
    const values = blank(fields);
    expect(Object.keys(missing(fields, values))).toEqual(["name"]);
    expect(missing(fields, { ...values, name: "Trà sen" })).toEqual({});
  });

  it("calls a record by its own name", () => {
    expect(titleOf({ id: "p1", name: "Nhà Gốm" })).toBe("Nhà Gốm");
    expect(titleOf({ id: "p1" })).toBe("p1");
    expect(titleOf(undefined)).toBe("");
  });

  it("marks what the shop works out as read-only and the sizes as sizes", () => {
    const specs = fieldsOf("products");
    for (const key of ["sizes_low", "sizes_out", "stock", "rating", "review_count"]) {
      expect(specs.find((f) => f.key === key)?.readOnly).toBe(true);
    }
    expect(specs.find((f) => f.key === "sizes")?.type).toBe("sizes");
    expect(specs.find((f) => f.key === "sizes")?.readOnly).toBeUndefined();
    expect(columnsFromFields(specs).map((c) => c.key)).not.toContain("sizes");
  });

  it("starts a new shoe as a draft with no sizes and no counts", () => {
    const start = blank(fieldsOf("products"));
    expect(start.status).toBe("draft");
    expect(start.sizes).toEqual([]);
    expect(start.pairs).toEqual({});
    expect(blank(fieldsOf("reviews")).status).toBe("");
  });
});

describe("what a form sends", () => {
  const specs = fieldsOf("products");
  const shoe: Row = {
    id: "tc-shoe", name: "Tc Shoe", price: 3290000, line: "samba", status: "live",
    sizes: ["38", "39", "40"], sizes_low: ["39"], sizes_out: ["40"], stock: 6, rating: 4.5, review_count: 2,
    pairs: { "38": 4, "39": 2, "40": 0 },
  };

  it("sends a new shoe's filled fields and its counts, with no id and nothing the shop derives", () => {
    const body = bodyOf(specs, {
      ...blank(specs), name: "Samba OG", price: 3290000, status: "live",
      sizes: ["38", "39"], pairs: { "38": "4", "39": "0" },
    });
    expect(body).toEqual({ name: "Samba OG", price: 3290000, status: "live", pairs: { "38": 4, "39": 0 } });
  });

  it("sends a change of the price alone as the price alone — no counts, no sizes", () => {
    expect(bodyOf(specs, { ...valuesOf(shoe), price: 3490000 }, shoe)).toEqual({ price: 3490000 });
    expect(bodyOf(specs, valuesOf(shoe), shoe)).toEqual({});
  });

  it("sends the counts with the ones last read when one is touched", () => {
    const body = bodyOf(specs, { ...valuesOf(shoe), pairs: { "38": "9", "39": "2", "40": "0" } }, shoe);
    expect(body).toEqual({
      pairs: { "38": 9, "39": 2, "40": 0 },
      seen_pairs: { "38": 4, "39": 2, "40": 0 },
    });
  });

  it("leaves a size out of the counts when it is taken off", () => {
    const body = bodyOf(specs, { ...valuesOf(shoe), sizes: ["38", "40"] }, shoe);
    expect(body.pairs).toEqual({ "38": 4, "40": 0 });
    expect(body).not.toHaveProperty("sizes");
  });

  it("empties a cleared number as null and a cleared text as empty", () => {
    const reviews = fieldsOf("reviews");
    const review: Row = { id: "r-1", product: "a", name: "Lan", rating: 4, body: "Good" };
    expect(bodyOf(reviews, { ...valuesOf(review), rating: "", body: "" }, review)).toEqual({ rating: null, body: "" });
  });

  it("refuses a size with no count, and a count that is not a whole number", () => {
    const added = { ...valuesOf(shoe), sizes: ["38", "39", "40", "41", "42"], pairs: { "38": "4", "39": "2.5", "40": "0", "41": "", "42": "0" } };
    expect(stockErrors(added, shoe)).toEqual({ "pairs.39": WHOLE_COUNT, "pairs.41": EMPTY_COUNT });
    expect(stockErrors(valuesOf(shoe), shoe)).toEqual({});
  });
});

describe("what Save and Delete report", () => {
  it("creates a record without sending an id, and returns the one the shop made", async () => {
    shop.create.mockResolvedValueOnce(reply(201, { id: "samba-og-new", name: "Samba OG New" }));
    const out = await saveRow("products", { name: "Samba OG New", price: 1 });
    expect(shop.create).toHaveBeenCalledWith("products", { name: "Samba OG New", price: 1 });
    expect(out).toEqual({ ok: true, value: { id: "samba-og-new", name: "Samba OG New" } });
  });

  it("changes a record by the id in its address, never in its body", async () => {
    shop.update.mockResolvedValueOnce(reply(200, { id: "a", price: 2 }));
    await saveRow("products", { id: "a", price: 2 });
    expect(shop.update).toHaveBeenCalledWith("products", "a", { price: 2 });
    expect(shop.create).not.toHaveBeenCalled();
  });

  const refusals: [string, number, unknown, object][] = [
    ["fields at fault", 422, { error: "invalid", message: "x", errors: { price: "Price cannot be below 0" } }, { kind: "fields", errors: { price: "Price cannot be below 0" } }],
    ["a count an order moved, naming the size", 409, { error: "stock_conflict", message: TAKEN, errors: { "pairs.38": TAKEN } }, { kind: "fields", errors: { "pairs.38": TAKEN } }],
    ["the shop's own sentence", 409, { error: "conflict", message: "Orders refer to this shoe. Set it to draft instead." }, { kind: "message", message: "Orders refer to this shoe. Set it to draft instead." }],
    ["a picture that is too large", 413, { error: "payload_too_large", message: "That picture is too large (5 MB at most)." }, { kind: "message", message: "That picture is too large (5 MB at most)." }],
    ["no session", 401, { error: "unauthenticated", message: "Sign in." }, { kind: "session" }],
    ["a customer's session", 403, { error: "forbidden", message: "No." }, { kind: "session" }],
    ["no answer", 0, undefined, { kind: "unreachable" }],
    ["a server fault", 500, { error: "internal", message: "Something went wrong." }, { kind: "unreachable" }],
    ["an answer that is not a record", 200, "nothing", { kind: "unreachable" }],
  ];
  it.each(refusals)("reports %s", async (_name, status, body, expected) => {
    shop.update.mockResolvedValueOnce(reply(status, body));
    expect(await saveRow("products", { id: "a", price: -1 })).toEqual({ ok: false, ...expected });
  });

  it("never rejects: a transport that throws is a shop that cannot be reached", async () => {
    shop.create.mockRejectedValueOnce(new Error("boom"));
    expect(await saveRow("lines", { name: "x" })).toEqual({ ok: false, kind: "unreachable" });
  });

  it("deletes, and passes the shop's refusal on as a sentence", async () => {
    shop.remove.mockResolvedValueOnce(reply(204));
    expect(await deleteRow("reviews", "r-1")).toEqual({ ok: true, value: null });
    shop.remove.mockResolvedValueOnce(reply(409, { error: "conflict", message: "3 shoes use this. Move them first." }));
    expect(await deleteRow("lines", "samba")).toEqual({ ok: false, kind: "message", message: "3 shoes use this. Move them first." });
  });

  it("reads the shoe again when an order moved a count, so the row can show what stands", async () => {
    shop.setPairs.mockResolvedValueOnce(reply(409, { error: "conflict", message: TAKEN }));
    shop.read.mockResolvedValueOnce(reply(200, { id: "tc", pairs: { "38": 2 } }));
    const out = await saveCount("tc", { size: "38", pairs: 6, seen_pairs: 4 });
    expect(shop.setPairs).toHaveBeenCalledWith("tc", { size: "38", pairs: 6, seen_pairs: 4 });
    expect(out).toEqual({ ok: false, kind: "stale", message: TAKEN, record: { id: "tc", pairs: { "38": 2 } } });
  });

  it("keeps the address a picture was stored at, and its refusal", async () => {
    const file = new File(["x"], "a.png", { type: "image/png" });
    shop.upload.mockResolvedValueOnce(reply(201, { url: "/api/v1/uploads/abc.png" }));
    expect(await uploadFile(file)).toEqual({ ok: true, value: "/api/v1/uploads/abc.png" });
    shop.upload.mockResolvedValueOnce(reply(415, { error: "unsupported_media_type", message: "Choose a JPEG, PNG or WebP picture." }));
    expect(await uploadFile(file)).toEqual({ ok: false, kind: "message", message: "Choose a JPEG, PNG or WebP picture." });
    shop.upload.mockResolvedValueOnce(reply(201, {}));
    expect(await uploadFile(file)).toEqual({ ok: false, kind: "unreachable" });
  });

  it("never gives two things the same key", () => {
    expect(newId("things")).not.toBe(newId("things"));
  });
});

// WHAT A FIELD IS, READ OFF ITS ROWS.
//
// The form run bd4d1058 shipped: "Status" as a box to type `live` into,
// "Photograph" as a box holding a file path, "Backdrop colour" as a box
// holding `var(--swatch-2)`. Each of those is decidable from the data, and
// these are the rows that form was drawn from, in miniature.
describe("a field read for what it is", () => {
  const rows: Row[] = Array.from({ length: 8 }, (_, i) => ({
    id: `p${i}`,
    name: `Garment ${i}`,
    image: `/images/products-${i}.jpg`,
    status: i % 3 === 0 ? "draft" : "live",
    backdrop: `var(--swatch-${(i % 4) + 1})`,
    fabric: `A fabric nobody else has, number ${i}`,
  }));

  const specs = () => {
    schema.collections.garments = {
      fields: [
        { key: "name", type: "text" },
        { key: "image", type: "text" },
        { key: "status", type: "text" },
        { key: "backdrop", type: "text" },
        { key: "fabric", type: "text" },
        { key: "swatch", type: "text", hidden: true },
        { key: "size", type: "text", options: ["s", ["m", "Medium"]] },
      ],
    };
    setCollection("garments", rows);
    return fieldsOf("garments");
  };

  it("makes a choice of a text field whose rows share a few values", () => {
    const status = specs().find((f) => f.key === "status");
    expect(status?.options?.map(([v]) => v).sort()).toEqual(["draft", "live"]);
    expect(specs().find((f) => f.key === "name")?.options).toBeUndefined();
    expect(specs().find((f) => f.key === "fabric")?.options).toBeUndefined();
  });

  it("sees that a column of image paths is a picture", () => {
    expect(specs().find((f) => f.key === "image")?.type).toBe("image");
  });

  it("leaves the design's own plumbing off the form", () => {
    const keys = specs().map((f) => f.key);
    expect(keys).not.toContain("backdrop");
    expect(keys).not.toContain("swatch");
  });

  it("believes the schema when it names the choices", () => {
    expect(specs().find((f) => f.key === "size")?.options).toEqual([["s", "S"], ["m", "Medium"]]);
  });
});
