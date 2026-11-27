/**
 * THE MANAGEMENT AREA'S WRITES — create, change and delete a record, send a
 * picture, change one size's count — and what each answer means to a screen.
 *
 * ONE TRANSPORT, ONE FILE TO SWAP. Every call goes through `transport`, which
 * answers with a plain {status, body} and never throws; status 0 is "no
 * answer came". Today it is the pretend shop in lib/mock; the backend step
 * points it at src/lib/api.ts (POST, PUT, DELETE, PATCH, the upload) and
 * deletes the mock. Nothing below it changes: what a 422 or a 409 means to a
 * person is decided here, once.
 *
 *   fields       the shop refused named fields (422, or a 409 that names a size)
 *   message      the shop's own sentence, to show as it is (409, 413, 415)
 *   stale        a count moved under the person (409 on one size): the sentence
 *                and the record as the shop has it now
 *   session      the session ended (401, 403): sign in again
 *   unreachable  no answer, a 5xx, anything unexpected
 *
 * No call is ever repeated by this file: a save is not idempotent.
 */
import type { Row } from "../data";
import transport from "./mock/manage-shoes-lines-categories-reviews-and-orders";

/** What a transport hands back. Status 0: no answer came. */
export interface Wire {
  status: number;
  body?: unknown;
}

export type Body = Record<string, unknown>;

export interface SizeChange {
  size: string;
  pairs: number;
  seen_pairs: number;
}

export interface Transport {
  create(collection: string, body: Body): Promise<Wire>;
  update(collection: string, id: string, body: Body): Promise<Wire>;
  remove(collection: string, id: string): Promise<Wire>;
  read(collection: string, id: string): Promise<Wire>;
  upload(file: File): Promise<Wire>;
  setPairs(productId: string, change: SizeChange): Promise<Wire>;
}

/** The shop's answer to one size's count: the size, its new count, the shoe's redrawn lists. */
export interface SizeAnswer {
  size: string;
  pairs: number;
  sizes: string[];
  sizes_low: string[];
  sizes_out: string[];
  stock: number;
}

export type Failure =
  | { kind: "fields"; errors: Record<string, string> }
  | { kind: "message"; message: string }
  | { kind: "stale"; message: string; record: Row | null }
  | { kind: "session" }
  | { kind: "unreachable" };

export type Outcome<T> = { ok: true; value: T } | ({ ok: false } & Failure);

function envelope(body: unknown): { message: string; errors: Record<string, string> } {
  const env = (typeof body === "object" && body !== null ? body : {}) as Record<string, unknown>;
  const errors = typeof env.errors === "object" && env.errors !== null ? (env.errors as Record<string, string>) : {};
  return { message: typeof env.message === "string" ? env.message : "", errors };
}

/** What a refusal stands for. A 5xx, no answer, or a refusal with no sentence to show is "unreachable". */
export function failureOf(wire: Wire): Failure {
  const { status } = wire;
  if (status === 401 || status === 403) return { kind: "session" };
  const { message, errors } = envelope(wire.body);
  if ((status === 422 || status === 400 || status === 409) && Object.keys(errors).length > 0) {
    return { kind: "fields", errors };
  }
  if (status >= 400 && status < 500 && message) return { kind: "message", message };
  return { kind: "unreachable" };
}

async function ask(run: () => Promise<Wire>): Promise<Wire> {
  try {
    return await run();
  } catch {
    return { status: 0 };
  }
}

function settle<T>(wire: Wire, take: (body: unknown) => T | null): Outcome<T> {
  if (wire.status >= 200 && wire.status < 300) {
    const value = take(wire.body);
    if (value !== null) return { ok: true, value };
    return { ok: false, kind: "unreachable" };
  }
  return { ok: false, ...failureOf(wire) };
}

const asRow = (body: unknown): Row | null => {
  const row = body as Row | undefined;
  return row && typeof row === "object" && typeof row.id === "string" ? row : null;
};

/** POST. The shop makes the id; none is ever sent. */
export async function createRecord(collection: string, body: Body): Promise<Outcome<Row>> {
  return settle(await ask(() => transport.create(collection, body)), asRow);
}

/** PUT. Only the fields in `body` change; the id is the address. */
export async function updateRecord(collection: string, id: string, body: Body): Promise<Outcome<Row>> {
  return settle(await ask(() => transport.update(collection, id, body)), asRow);
}

export async function removeRecord(collection: string, id: string): Promise<Outcome<null>> {
  const wire = await ask(() => transport.remove(collection, id));
  return wire.status === 204 || wire.status === 200 ? { ok: true, value: null } : { ok: false, ...failureOf(wire) };
}

/** A picture to the shop; the answer is the address to keep on the record. */
export async function uploadPicture(file: File): Promise<Outcome<string>> {
  const wire = await ask(() => transport.upload(file));
  return settle(wire, (body) => {
    const url = (body as { url?: unknown } | undefined)?.url;
    return typeof url === "string" && url !== "" ? url : null;
  });
}

/**
 * One size's count on its own, with the count the person last saw. A 409 means
 * an order moved it: the shoe is read again so the row can show what stands.
 */
export async function setPairs(productId: string, change: SizeChange): Promise<Outcome<SizeAnswer>> {
  const wire = await ask(() => transport.setPairs(productId, change));
  if (wire.status === 409) {
    const fresh = await ask(() => transport.read("products", productId));
    const { message } = envelope(wire.body);
    return { ok: false, kind: "stale", message: message || "Pairs changed. Check the number and save again.", record: fresh.status === 200 ? asRow(fresh.body) : null };
  }
  return settle(wire, (body) => {
    const a = body as Partial<SizeAnswer> | undefined;
    return a && typeof a.size === "string" && typeof a.pairs === "number" ? (a as SizeAnswer) : null;
  });
}
