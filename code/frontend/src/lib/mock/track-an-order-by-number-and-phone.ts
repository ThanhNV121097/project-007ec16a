import { ApiError } from "../api";
import type { LookupAnswer, LookupBody } from "../track-an-order-by-number-and-phone";

/**
 * THE SHOP'S ANSWER TO POST /v1/order-lookups, for the UI stage only. Deleted
 * by the backend step, which points the client's `send` at the real route.
 *
 * It behaves the way the shop is specified to, so every state of the page can
 * be reached with data and nothing else: the sample order DOI-24817 (phone
 * 0938 410 552) is found; any other pair is the one 404; the sixth miss within
 * 15 minutes is a 429 whether or not the pair is right; and the number 50000
 * is a shop that errors (500). A page control never fakes any of these.
 */
const SAMPLE = { number: "DOI-24817", placed_at: "2026-11-24T09:12:00+07:00", phone: "0938410552" };
const WINDOW_MS = 15 * 60 * 1000;
const LIMIT = 5;
const SLOW_MS = 700;

const misses: { at: number; number: string }[] = [];

const digits = (s: string) => s.replace(/\D/g, "");
const phoneKey = (s: string) => {
  const d = digits(s);
  return d.startsWith("84") ? `0${d.slice(2)}` : d;
};

function locked(number: string): boolean {
  const recent = misses.filter((m) => Date.now() - m.at < WINDOW_MS);
  return recent.length >= LIMIT || recent.filter((m) => m.number === number).length >= LIMIT;
}

function token(): string {
  return Array.from({ length: 3 }, () => Math.random().toString(36).slice(2, 10).padEnd(8, "0")).join("");
}

export async function postLookup({ number, phone }: LookupBody): Promise<LookupAnswer> {
  await new Promise((done) => setTimeout(done, SLOW_MS));
  const key = digits(number);
  if (key === "50000") throw new ApiError(500, "internal", "Something went wrong. Please try again.");
  if (locked(key)) throw new ApiError(429, "rate_limited", "Too many tries.");
  if (key === digits(SAMPLE.number) && phoneKey(phone) === SAMPLE.phone) {
    return { id: "doi-24817", number: SAMPLE.number, placed_at: SAMPLE.placed_at, token: token() };
  }
  misses.push({ at: Date.now(), number: key });
  throw new ApiError(404, "not_found", "Not found.");
}
