/**
 * The project's single API client. Every story that talks to the backend
 * imports this file; a second client would be a second set of timeouts, error
 * shapes and retry rules.
 *
 * Every request runs in the browser. The base is VITE_API_URL, or "/api" when
 * it is not set (an EMPTY value is not unset and is used as given); a request
 * goes to `${base}/v1/<path>`. In a deployment the edge proxy strips /api, and
 * the call is same-origin, so a session cookie travels with it.
 */

/** What a request may carry in its query string. `undefined` is left out. */
export type Params = Record<string, string | number | boolean | undefined>;

/** A failed request: the server's error envelope, or a client-side "network". */
export class ApiError extends Error {
  /** The HTTP status; 0 when no answer came (no connection, or a timeout). */
  readonly status: number;
  /** The envelope's `error` word; "network" when no answer came. */
  readonly code: string;
  /** The envelope's `errors`: input name → sentence, on a 400 or 422. */
  readonly errors?: Record<string, string>;

  constructor(status: number, code: string, message: string, errors?: Record<string, string>) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
    this.errors = errors;
  }
}

const TIMEOUT_MS = 15_000;
const GENERIC = "Something went wrong. Please try again.";

export function apiBase(): string {
  const base: string = import.meta.env.VITE_API_URL ?? "/api";
  return base.replace(/\/+$/, "");
}

/** `a=1&b=x`, a comma left as it is so `category=women,kids` reads as written. */
function queryString(params?: Params): string {
  const parts: string[] = [];
  for (const [key, value] of Object.entries(params ?? {})) {
    if (value === undefined) continue;
    parts.push(`${encodeURIComponent(key)}=${encodeURIComponent(String(value)).replace(/%2C/gi, ",")}`);
  }
  return parts.length > 0 ? `?${parts.join("&")}` : "";
}

/** The error an answer that is not 2xx stands for: the envelope when it is one. */
function failure(status: number, body: unknown): ApiError {
  const env = (typeof body === "object" && body !== null ? body : {}) as Record<string, unknown>;
  const code = typeof env.error === "string" ? env.error : "internal";
  const message = typeof env.message === "string" ? env.message : GENERIC;
  const errors = typeof env.errors === "object" && env.errors !== null ? (env.errors as Record<string, string>) : undefined;
  return new ApiError(status, code, message, errors);
}

/** Request headers a call may add to the ones every call carries (an order's `X-Order-Token`). */
export type ExtraHeaders = Record<string, string>;

async function send(url: string, signal: AbortSignal, extra?: ExtraHeaders): Promise<unknown> {
  let res: Response;
  try {
    res = await fetch(url, { credentials: "include", signal, headers: { Accept: "application/json", ...extra } });
  } catch {
    throw new ApiError(0, "network", "The server cannot be reached.");
  }
  const body: unknown = await res.json().catch(() => undefined);
  if (!res.ok) throw failure(res.status, body);
  if (body === undefined) throw failure(res.status, undefined);
  return body;
}

/** Runs one request and abandons it after `ms` (15 seconds unless said). The abort counts as a failure. */
async function timed<T>(run: (signal: AbortSignal) => Promise<T>, ms: number = TIMEOUT_MS): Promise<T> {
  const abort = new AbortController();
  let timer: ReturnType<typeof setTimeout> | undefined;
  const timedOut = new Promise<never>((_, reject) => {
    timer = setTimeout(() => {
      abort.abort();
      reject(new ApiError(0, "network", "The server took too long to answer."));
    }, ms);
  });
  try {
    return await Promise.race([run(abort.signal), timedOut]);
  } finally {
    clearTimeout(timer);
  }
}

/** One GET try, abandoned after 15 seconds. */
async function attempt(url: string, extra?: ExtraHeaders): Promise<unknown> {
  return timed((signal) => send(url, signal, extra));
}

/** No answer, or a 5xx, may pass on a second try. A 404 or any other 4xx is an answer. */
function retriable(e: unknown): boolean {
  return e instanceof ApiError && (e.status === 0 || e.status >= 500);
}

const inflight = new Map<string, Promise<unknown>>();

/** Identical GETs asked at the same moment (twelve cards, one line) share a request. */
function shared<T>(key: string, run: () => Promise<T>): Promise<T> {
  const open = inflight.get(key);
  if (open) return open as Promise<T>;
  const started = run().finally(() => inflight.delete(key));
  inflight.set(key, started);
  return started;
}

/**
 * GET `${base}/v1/<path>`. A failed GET is tried once more; a 404 never is.
 * Resolves with the parsed body; rejects with an ApiError. `headers` are added to
 * the request; two reads share a request only when their address AND their
 * headers are the same, so one caller's credential never answers another's.
 */
async function get<T>(path: string, params?: Params, headers?: ExtraHeaders): Promise<T> {
  const url = `${apiBase()}/v1/${path.replace(/^\/+/, "")}${queryString(params)}`;
  const key = headers ? `${url}\n${JSON.stringify(Object.entries(headers).sort())}` : url;
  return shared(key, async () => {
    try {
      return (await attempt(url, headers)) as T;
    } catch (e) {
      if (!retriable(e)) throw e;
      return (await attempt(url, headers)) as T;
    }
  });
}

type Write = "POST" | "PUT" | "PATCH" | "DELETE";

/** A form is sent as it is, so the browser writes its own multipart Content-Type and boundary. */
const isForm = (body: unknown): body is FormData => typeof FormData !== "undefined" && body instanceof FormData;

<<<<<<< refs/heads/feature/manage-shoes-lines-categories-reviews-and-orders-be
async function sendWrite(method: Write, url: string, body: unknown, signal: AbortSignal): Promise<unknown> {
  const headers: Record<string, string> = { Accept: "application/json" };
  if (body !== undefined && !isForm(body)) headers["Content-Type"] = "application/json";
=======
async function sendWrite(method: Write, url: string, body: unknown, signal: AbortSignal, extra?: ExtraHeaders): Promise<unknown> {
  const headers: ExtraHeaders = { Accept: "application/json", ...extra };
  if (body !== undefined) headers["Content-Type"] = "application/json";
>>>>>>> refs/heads/main
  let res: Response;
  try {
    res = await fetch(url, {
      method, credentials: "include", signal, headers,
      body: body === undefined ? undefined : isForm(body) ? body : JSON.stringify(body),
    });
  } catch {
    throw new ApiError(0, "network", "The server cannot be reached.");
  }
  // An answer with nothing to say (sign-out, a removal) is a success, not a failure.
  if (res.status === 204) return undefined;
  const parsed: unknown = await res.json().catch(() => undefined);
  if (!res.ok) throw failure(res.status, parsed);
  if (parsed === undefined) throw failure(res.status, undefined);
  return parsed;
}

<<<<<<< refs/heads/feature/manage-shoes-lines-categories-reviews-and-orders-be
async function write<T>(method: Write, path: string, body?: unknown, ms: number = TIMEOUT_MS): Promise<T> {
  const url = `${apiBase()}/v1/${path.replace(/^\/+/, "")}`;
  return (await timed((signal) => sendWrite(method, url, body, signal), ms)) as T;
=======
async function write<T>(method: Write, path: string, body?: unknown, headers?: ExtraHeaders): Promise<T> {
  const url = `${apiBase()}/v1/${path.replace(/^\/+/, "")}`;
  return (await timed((signal) => sendWrite(method, url, body, signal, headers))) as T;
>>>>>>> refs/heads/main
}

/**
 * POST `${base}/v1/<path>` with a JSON body, or none. Never tried again: a POST
 * is not idempotent. A 204 resolves undefined; any other failure rejects with
 * an ApiError, a refusal carrying the server's envelope. `headers` are added to
 * the request; with no body the call sends none and no content type.
 */
async function post<T = void>(path: string, body?: unknown, headers?: ExtraHeaders): Promise<T> {
  return write<T>("POST", path, body, headers);
}

/** PUT `${base}/v1/<path>` with a JSON body. Never tried again; failures as post. */
async function put<T = void>(path: string, body?: unknown): Promise<T> {
  return write<T>("PUT", path, body);
}

/** DELETE `${base}/v1/<path>`. A 204 resolves undefined; never tried again; failures as post. */
async function del(path: string): Promise<void> {
  return write<void>("DELETE", path);
}

/** PATCH `${base}/v1/<path>` with a JSON body. Never tried again; failures as post. */
async function patch<T = void>(path: string, body?: unknown): Promise<T> {
  return write<T>("PATCH", path, body);
}

/** A picture is slower than a record: its POST is given 30 seconds, not 15. */
const UPLOAD_TIMEOUT_MS = 30_000;

/**
 * POST a form (a file) to `${base}/v1/<path>`. Never tried again; failures as
 * post, and the 30 seconds an upload on a phone's connection may need.
 */
async function upload<T = void>(path: string, form: FormData): Promise<T> {
  return write<T>("POST", path, form, UPLOAD_TIMEOUT_MS);
}

export const api = { get, post, put, patch, delete: del, upload };
