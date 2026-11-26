/**
 * THE MANAGEMENT AREA'S WRITES AS THE SHOP ANSWERS THEM.
 *
 * The one place the six calls of src/lib/manage-shoes-lines-categories-reviews-
 * and-orders.ts meet src/lib/api.ts: POST, PUT, DELETE, a picture, one size's
 * count (PATCH) and a read of one record. Each answers a plain {status, body}
 * and never throws; status 0 is "no answer came". An ApiError becomes the
 * answer the shop gave (its status, its envelope); anything else is "no answer".
 *
 * Nothing here is repeated by itself: a save is not idempotent, and api.ts does
 * not retry a write. The reads go as the signed-in dealer (the session cookie),
 * so they carry what a visitor never sees.
 */
import { api, ApiError } from "./api";
import type { Transport, Wire } from "./manage-shoes-lines-categories-reviews-and-orders";

const record = (collection: string, id: string) => `${encodeURIComponent(collection)}/${encodeURIComponent(id)}`;

/** Runs a call; a success carries the status the shop gives it, a refusal its envelope. */
async function answer(success: number, run: () => Promise<unknown>): Promise<Wire> {
  try {
    return { status: success, body: await run() };
  } catch (e) {
    if (!(e instanceof ApiError)) return { status: 0 };
    return { status: e.status, body: { error: e.code, message: e.message, ...(e.errors ? { errors: e.errors } : {}) } };
  }
}

const transport: Transport = {
  create: (collection, body) => answer(201, () => api.post(encodeURIComponent(collection), body)),
  update: (collection, id, body) => answer(200, () => api.put(record(collection, id), body)),
  remove: (collection, id) => answer(204, () => api.delete(record(collection, id))),
  read: (collection, id) => answer(200, () => api.get(record(collection, id))),
  upload: (file) => {
    const form = new FormData();
    form.append("file", file);
    return answer(201, () => api.upload("uploads", form));
  },
  setPairs: (productId, change) => answer(200, () => api.patch(`${record("products", productId)}/sizes`, change)),
};

export default transport;
