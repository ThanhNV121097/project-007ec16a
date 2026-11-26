import { useEffect, useState, useSyncExternalStore, type Dispatch, type SetStateAction } from "react";
import type { OrderStatus } from "./shop";
import * as shop from "./mock/customer-accounts-sign-up-sign-in-my-orders";

/**
 * THE STORY'S API CLIENT: what the customer-account screens ask the shop.
 *
 * Today every call goes to the story's mock; the backend step points `shop`
 * at src/lib/api.ts (credentials: "include") and deletes the mock, and no
 * component changes. Who is signed in comes ONLY from GET /v1/auth/me:
 * nothing here keeps a session of its own, so a forged browser value opens
 * nothing, and every sign-in, sign-up and sign-out ends by asking again.
 */

export type Role = "customer" | "staff" | "admin";

export interface Me {
  email: string;
  name: string;
  role: Role;
}

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

/** Ten rows at a time (services.md 3.20). */
export const PAGE_SIZE = 10;

function fieldErrors(body: unknown): FieldErrors {
  const raw = (body as { errors?: unknown } | null)?.errors;
  const out: FieldErrors = {};
  if (raw && typeof raw === "object") {
    for (const [key, value] of Object.entries(raw)) if (typeof value === "string") out[key] = value;
  }
  return out;
}

/** A network failure and a 5xx are one thing to the customer: the shop could not be reached. */
async function ask<T>(run: () => Promise<shop.Reply>): Promise<Outcome<T>> {
  let answer: shop.Reply;
  try {
    answer = await run();
  } catch {
    return { kind: "unreachable" };
  }
  if (answer.status >= 200 && answer.status < 300) return { kind: "ok", value: answer.body as T };
  switch (answer.status) {
    case 401: return { kind: "unauthenticated" };
    case 403: return { kind: "forbidden" };
    case 404: return { kind: "notfound" };
    case 409: return { kind: "conflict", errors: fieldErrors(answer.body) };
    case 422: return { kind: "invalid", errors: fieldErrors(answer.body) };
    case 429: return { kind: "limited" };
    default: return { kind: "unreachable" };
  }
}

// ── who is signed in: GET /v1/auth/me, and nothing else ──────────────────────

export interface MeState {
  /** True until /v1/auth/me has answered once: draw nothing that depends on the visitor. */
  checking: boolean;
  me: Me | null;
}

let state: MeState = { checking: true, me: null };
let asked = false;
const listeners = new Set<() => void>();

function publish(next: MeState) {
  state = next;
  listeners.forEach((l) => l());
}

function subscribe(l: () => void) {
  listeners.add(l);
  return () => {
    listeners.delete(l);
  };
}

/** Asks /v1/auth/me again. A 401 or an unreachable shop is "nobody", and draws no error. */
export async function refreshMe(): Promise<void> {
  const out = await ask<Me>(() => shop.getMe());
  publish({ checking: false, me: out.kind === "ok" ? out.value : null });
}

/** The visitor as the shop knows them. Every component that reads it draws again when it changes. */
export function useMe(): MeState {
  const current = useSyncExternalStore(subscribe, () => state, () => state);
  useEffect(() => {
    if (asked) return;
    asked = true;
    void refreshMe();
  }, []);
  return current;
}

/** Where a signed-in person lands: a customer's account, the team's area. */
export const homeFor = (me: Me): string => (me.role === "customer" ? "/account" : "/admin");

// ── signing up, in and out ───────────────────────────────────────────────────

export async function customerSignUp(input: { name: string; email: string; phone: string; password: string }): Promise<Outcome<Me>> {
  const out = await ask<Me>(() => shop.signUp(input));
  if (out.kind === "ok") await refreshMe();
  return out;
}

export async function customerSignIn(email: string, password: string): Promise<Outcome<Me>> {
  const out = await ask<Me>(() => shop.signIn(email, password));
  if (out.kind === "ok") await refreshMe();
  return out;
}

/** Ends the session, then asks who is signed in: the shop's answer decides, not this page. */
export async function customerSignOut(): Promise<void> {
  await ask<null>(() => shop.signOut());
  await refreshMe();
}

// ── the account's own data ───────────────────────────────────────────────────

export const getAccount = () => ask<Account>(() => shop.getAccount());
export const saveAccount = (details: Details) => ask<Account>(() => shop.putAccount(details));
export const listOrders = (page: number) => ask<OrdersPage>(() => shop.listOrders(page, PAGE_SIZE));
export const addOrder = (number: string, phone: string) => ask<OrderRow>(() => shop.addOrder(number, phone));

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
