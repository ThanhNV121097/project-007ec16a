/**
 * UI stage only: the shop's staff answers, as the API will give them
 * (services.md §3.26, §3.27). The backend stage deletes this file and
 * src/lib/staff.ts calls src/lib/api.ts instead.
 *
 * It answers as the shop would: the list is slow to come, a save checks its
 * fields and says every one at fault, the admin's own row is refused, and a
 * browser with no network gets no answer at all.
 */
export interface MockAccount {
  id: string;
  name: string;
  email: string;
  role: "admin" | "staff";
  is_active: boolean;
}

/** What a non-2xx answer carries: the status and the shop's body. */
export class MockFault extends Error {
  constructor(
    readonly status: number,
    readonly body: { error: string; message: string; errors?: Record<string, string> },
  ) {
    super(body.message);
  }
}

let accounts: MockAccount[] = [
  { id: "0b1c7a52-1f0e-4a52-9d57-0f0d3c1a0001", name: "Shop owner", email: "owner@example.com", role: "admin", is_active: true },
  { id: "0b1c7a52-1f0e-4a52-9d57-0f0d3c1a0002", name: "Minh Nguyen", email: "minh@example.com", role: "staff", is_active: true },
  { id: "0b1c7a52-1f0e-4a52-9d57-0f0d3c1a0003", name: "Hoa Pham", email: "hoa@example.com", role: "staff", is_active: false },
];
const customerEmails = ["customer@example.com"];
let counter = 3;

const wait = (ms: number) => new Promise<void>((done) => setTimeout(done, ms));
const length = (s: string) => [...s].length;

async function reach(ms: number): Promise<void> {
  await wait(ms);
  if (typeof navigator !== "undefined" && navigator.onLine === false) {
    throw new MockFault(0, { error: "network", message: "We cannot reach the server." });
  }
}

function refuse(errors: Record<string, string>): never {
  throw new MockFault(422, { error: "validation_failed", message: "Please correct the fields marked.", errors });
}

function fieldErrors(body: Record<string, unknown>, adding: boolean, selfId?: string): Record<string, string> {
  const errors: Record<string, string> = {};
  if ("name" in body || adding) {
    const name = String(body.name ?? "").trim();
    if (!name) errors.name = "A name is needed";
    else if (length(name) > 100) errors.name = "The name can have at most 100 characters.";
  }
  if (adding) {
    const email = String(body.email ?? "").trim().toLowerCase();
    const taken = [...accounts.filter((a) => a.id !== selfId).map((a) => a.email), ...customerEmails];
    if (!email) errors.email = "An email address is needed";
    else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email) || length(email) > 200) errors.email = "That does not look like an email address.";
    else if (taken.includes(email)) errors.email = "That email already belongs to an account.";
  }
  if ("password" in body || adding) {
    const n = length(String(body.password ?? ""));
    if (n < 10) errors.password = "The password needs at least 10 characters.";
    else if (n > 100) errors.password = "The password can have at most 100 characters.";
  }
  if ("role" in body) errors.role = "The role cannot be changed.";
  return errors;
}

function target(id: string): MockAccount {
  const found = accounts.find((a) => a.id === id);
  if (found?.role === "admin") {
    throw new MockFault(409, { error: "conflict", message: "The admin account cannot be switched off, changed or removed." });
  }
  if (!found) throw new MockFault(404, { error: "not_found", message: "That account was not found." });
  return found;
}

export async function mockList(): Promise<{ items: MockAccount[]; total: number; page: number; limit: number }> {
  await reach(700);
  return { items: accounts.map((a) => ({ ...a })), total: accounts.length, page: 1, limit: 100 };
}

export async function mockCreate(body: { name: string; email: string; password: string }): Promise<MockAccount> {
  await reach(500);
  const errors = fieldErrors(body, true);
  if (Object.keys(errors).length > 0) refuse(errors);
  counter += 1;
  const made: MockAccount = {
    id: `0b1c7a52-1f0e-4a52-9d57-0f0d3c1a${String(counter).padStart(4, "0")}`,
    name: body.name.trim(),
    email: body.email.trim().toLowerCase(),
    role: "staff",
    is_active: true,
  };
  accounts = [...accounts, made];
  return { ...made };
}

export async function mockUpdate(id: string, body: { name?: string; is_active?: boolean; password?: string }): Promise<MockAccount> {
  await reach(500);
  const found = target(id);
  const errors = fieldErrors(body, false, id);
  if (Object.keys(errors).length > 0) refuse(errors);
  const next: MockAccount = {
    ...found,
    name: body.name !== undefined ? body.name.trim() : found.name,
    is_active: body.is_active ?? found.is_active,
  };
  accounts = accounts.map((a) => (a.id === id ? next : a));
  return { ...next };
}

export async function mockRemove(id: string): Promise<void> {
  await reach(500);
  target(id);
  accounts = accounts.filter((a) => a.id !== id);
}
