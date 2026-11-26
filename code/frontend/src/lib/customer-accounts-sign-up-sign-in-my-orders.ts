import { useEffect, useState, type Dispatch, type SetStateAction } from "react";
import { ApiError, api } from "./api";
import { refreshSession, signOut, useAuth, type Session } from "./session";
import type { OrderStatus } from "./shop";

/**
 * THE STORY'S API CLIENT: what the customer-account screens ask the shop,
 * through src/lib/api.ts (the session cookie travels with every call).
 *
 * Who is signed in comes ONLY from GET /v1/auth/me, and the one cache of that
 * answer is src/lib/session.ts's: nothing here keeps a session of its own, so a
 * forged browser value opens nothing, and every sign-in, sign-up, save and
 * sign-out ends by asking again (refreshSession).
 */

export type Me = Session;
export type Role = Session["role"];

export interface Account {
  name: string;
  email: string;
  phone: string;
  address: string;
  city: string;
}

export type Details = Pick<Account, "name" | "phone" | "address" | "city">;

/** What a list row and the add-an-order answer hold: never a phone, email, address or note. */
export interface OrderRow {
  id: string;
  number: string;
  placed_at: string;
  status: OrderStatus;
  total: number;
}

export interface OrdersPage {
  items: OrderRow[];
  total: number;
  page: number;
  limit: number;
}

export type FieldErrors = Record<string, string>;

/**
 * The shop's answer, sorted by what a screen does about it. For a sign-in,
 * "unauthenticated" (401) is a wrong email or password.
 */
export type Outcome<T> =
  | { kind: "ok"; value: T }
  | { kind: "invalid"; errors: FieldErrors }
  | { kind: "conflict"; errors: FieldErrors }
  | { kind: "unauthenticated" }
  | { kind: "forbidden" }
  | { kind: "notfound" }
  | { kind: "limited" }
  | { kind: "unreachable" };

/** Ten rows at a time (services 3.20). */
export const PAGE_SIZE = 10;

const isObject = (v: unknown): v is Record<string, unknown> => typeof v === "object" && v !== null;
const isText = (...values: unknown[]) => values.every((v) => typeof v === "string");

const isMe = (v: unknown): v is Me =>
  isObject(v) && isText(v.email, v.name) && (v.role === "customer" || v.role === "staff" || v.role === "admin");
const isAccount = (v: unknown): v is Account => isObject(v) && isText(v.name, v.email, v.phone, v.address, v.city);
const isRow = (v: unknown): v is OrderRow =>
  isObject(v) && isText(v.id, v.number, v.placed_at, v.status) && typeof v.total === "number";
const isPage = (v: unknown): v is OrdersPage =>
  isObject(v) && Array.isArray(v.items) && v.items.every(isRow) &&
  typeof v.total === "number" && typeof v.page === "number" && typeof v.limit === "number";

/**
 * A status is sorted into its kind ONLY when the answer carries the API's own
 * envelope code for it. Everything else (no answer, a 5xx, a 400, and a 401,
 * 403 or 404 whose code is `internal`: a gateway's page, a mis-routed /api) is
 * "unreachable", so a broken proxy never tells a customer "Wrong email or
 * password" or "No order matches that number and phone".
 */
function sorted(e: ApiError): Outcome<never> {
  const { status, code } = e;
  if (status === 401 && (code === "unauthenticated" || code === "invalid_credentials")) return { kind: "unauthenticated" };
  if (status === 403 && code === "forbidden") return { kind: "forbidden" };
  if (status === 404 && code === "not_found") return { kind: "notfound" };
  if (status === 409 && code === "conflict") return { kind: "conflict", errors: e.errors ?? {} };
  if (status === 422 && code === "invalid") return { kind: "invalid", errors: e.errors ?? {} };
  if (status === 429 && code === "rate_limited") return { kind: "limited" };
  return { kind: "unreachable" };
}

/** Runs one call and sorts what comes back; a 2xx body that is not the expected shape is "unreachable" too. */
async function ask<T>(run: () => Promise<unknown>, valid: (body: unknown) => body is T): Promise<Outcome<T>> {
  try {
    const body = await run();
    return valid(body) ? { kind: "ok", value: body } : { kind: "unreachable" };
  } catch (e) {
    return e instanceof ApiError ? sorted(e) : { kind: "unreachable" };
  }
}

// ── who is signed in: GET /v1/auth/me, and nothing else ────────────────────────────────────────

export interface MeState {
  /** True until /v1/auth/me has answered once: draw nothing that depends on the visitor. */
  checking: boolean;
  me: Me | null;
}

/** The visitor as the shop knows them (session.ts holds the one answer). */
export function useMe(): MeState {
  const { checking, session } = useAuth();
  return { checking, me: session };
}

/** Asks /v1/auth/me again. A 401 or an unreachable shop is "nobody", and draws no error. */
export const refreshMe = refreshSession;

/** Where a signed-in person lands: a customer's account, the team's area. */
export const homeFor = (me: Me): string => (me.role === "customer" ? "/account" : "/admin");

// ── signing up, in and out ────────────────────────────────────────────────────────────

export async function customerSignUp(input: { name: string; email: string; phone: string; password: string }): Promise<Outcome<Me>> {
  const out = await ask(() => api.post<unknown>("account/sign-up", input), isMe);
  if (out.kind === "ok") await refreshSession();
  return out;
}

export async function customerSignIn(email: string, password: string): Promise<Outcome<Me>> {
  const out = await ask(() => api.post<unknown>("auth/sign-in", { email, password }), isMe);
  if (out.kind === "ok") await refreshSession();
  return out;
}

/** Ends the session, then asks who is signed in: the shop's answer decides, not this page. */
export async function customerSignOut(): Promise<void> {
  await signOut();
  await refreshSession();
}

// ── the account's own data ──────────────────────────────────────────────────────────────

export const getAccount = () => ask(() => api.get<unknown>("account"), isAccount);

/** Saves the four details. On success the header is asked again, so a new name shows at once. */
export async function saveAccount(details: Details): Promise<Outcome<Account>> {
  const out = await ask(() => api.put<unknown>("account", details), isAccount);
  if (out.kind === "ok") await refreshSession();
  return out;
}

export const listOrders = (page: number) => ask(() => api.get<unknown>("account/orders", { page, limit: PAGE_SIZE }), isPage);
export const addOrder = (number: string, phone: string) => ask(() => api.post<unknown>("account/orders", { number, phone }), isRow);

/** The saved details of the signed-in customer; null for anyone else and until they arrive. */
export function useSavedDetails(): Account | null {
  const { me } = useMe();
  const customer = me?.role === "customer";
  const [saved, setSaved] = useState<Account | null>(null);
  useEffect(() => {
    if (!customer) return undefined;
    let live = true;
    void getAccount().then((out) => {
      if (live && out.kind === "ok") setSaved(out.value);
    });
    return () => {
      live = false;
    };
  }, [customer]);
  return customer ? saved : null;
}

/**
 * Fills a form's name, phone, address and city from the saved details when
 * they arrive, and only the fields still empty: what the visitor has already
 * typed is never replaced by a value that came in late.
 */
export function useSavedDetailsFill<F extends Details>(setForm: Dispatch<SetStateAction<F>>): void {
  const saved = useSavedDetails();
  useEffect(() => {
    if (!saved) return;
    setForm((f) => ({
      ...f,
      name: f.name || saved.name,
      phone: f.phone || saved.phone,
      address: f.address || saved.address,
      city: f.city || saved.city,
    }));
  }, [saved, setForm]);
}
