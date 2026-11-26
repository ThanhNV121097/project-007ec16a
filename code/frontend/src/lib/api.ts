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

async function send(url: string, signal: AbortSignal): Promise<unknown> {
  let res: Response;
  try {
    res = await fetch(url, { credentials: "include", signal, headers: { Accept: "application/json" } });
  } catch {
    throw new ApiError(0, "network", "The server cannot be reached.");
  }
  const body: unknown = await res.json().catch(() => undefined);
  if (!res.ok) throw failure(res.status, body);
  if (body === undefined) throw failure(res.status, undefined);
  return body;
}

/** Runs one request and abandons it after 15 seconds. The abort counts as a failure. */
async function timed<T>(run: (signal: AbortSignal) => Promise<T>): Promise<T> {
  const abort = new AbortController();
  let timer: ReturnType<typeof setTimeout> | undefined;
  const timedOut = new Promise<never>((_, reject) => {
    timer = setTimeout(() => {
      abort.abort();
      reject(new ApiError(0, "network", "The server took too long to answer."));
    }, TIMEOUT_MS);
  });
  try {
    return await Promise.race([run(abort.signal), timedOut]);
  } finally {
    clearTimeout(timer);
  }
}

/** One GET try, abandoned after 15 seconds. */
async function attempt(url: string): Promise<unknown> {
  return timed((signal) => send(url, signal));
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
 * Resolves with the parsed body; rejects with an ApiError.
 */
async function get<T>(path: string, params?: Params): Promise<T> {
  const url = `${apiBase()}/v1/${path.replace(/^\/+/, "")}${queryString(params)}`;
  return shared(url, async () => {
    try {
      return (await attempt(url)) as T;
    } catch (e) {
      if (!retriable(e)) throw e;
      return (await attempt(url)) as T;
    }
  });
}

type Write = "POST" | "PUT" | "DELETE";

async function sendWrite(method: Write, url: string, body: unknown, signal: AbortSignal): Promise<unknown> {
  const headers: Record<string, string> = { Accept: "application/json" };
  if (body !== undefined) headers["Content-Type"] = "application/json";
  let res: Response;
  try {
    res = await fetch(url, {
      method, credentials: "include", signal, headers,
      body: body === undefined ? undefined : JSON.stringify(body),
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

async function write<T>(method: Write, path: string, body?: unknown): Promise<T> {
  const url = `${apiBase()}/v1/${path.replace(/^\/+/, "")}`;
  return (await timed((signal) => sendWrite(method, url, body, signal))) as T;
}

/**
 * POST `${base}/v1/<path>` with a JSON body, or none. Never tried again: a POST
 * is not idempotent. A 204 resolves undefined; any other failure rejects with
 * an ApiError, a refusal carrying the server's envelope.
 */
async function post<T = void>(path: string, body?: unknown): Promise<T> {
  return write<T>("POST", path, body);
}

/** PUT `${base}/v1/<path>` with a JSON body. Never tried again; failures as post. */
async function put<T = void>(path: string, body?: unknown): Promise<T> {
  return write<T>("PUT", path, body);
}

/** DELETE `${base}/v1/<path>`. A 204 resolves undefined; never tried again; failures as post. */
async function del(path: string): Promise<void> {
  return write<void>("DELETE", path);
}

export const api = { get, post, put, delete: del };
