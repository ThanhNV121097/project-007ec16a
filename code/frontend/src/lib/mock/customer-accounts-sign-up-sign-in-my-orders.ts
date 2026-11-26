/**
 * THE MOCK for "Customer accounts: sign up, sign in, my orders", standing in
 * for /v1/auth/me, /v1/auth/sign-in, /v1/auth/sign-out and /v1/account/*.
 * The backend step deletes this module and points
 * src/lib/customer-accounts-sign-up-sign-in-my-orders.ts (its only importer)
 * at src/lib/api.ts. Every answer has the status and body of
 * docs/architecture/services.md sections 3.7 to 3.9 and 3.18 to 3.21; a
 * network failure is a thrown error.
 *
 * Nothing is stored outside this page's memory: no password is kept (the one
 * rule below only looks at what was typed), and nothing is written to the
 * browser, so a reload signs the sample customer out.
 *
 * HOW A SAMPLE IS REACHED, by what is typed (no control on the page):
 *   sign in as      lan@example.com   customer Lan Nguyen, 2 orders
 *                   new@example.com   customer Thu Pham, no orders
 *                   many@example.com  customer Hoa Tran, 12 orders, address saved
 *                   slow@example.com  customer Tuan Vo: sign-in and sign-up take 3 s;
 *                                     the first orders answer takes 3 s and fails (500)
 *                   minh@example.com  staff Minh;  dealer@example.com  admin Dealer
 *                   any password works unless it starts with "wrong"
 *   an email starting with "down"  the shop cannot be reached
 *   an email starting with "limit" 429
 *   saved details   name "Slow…" takes 2 s; name "Down…" cannot be reached;
 *                   phone 0900 000 000 is refused with 422
 *   add an order    DOI-24818 with 0912 345 678 is the one guest order that exists;
 *                   number 24801 cannot be reached; five wrong tries lock it (429)
 */

export interface Reply {
  status: number;
  body: unknown;
}

type Role = "customer" | "staff" | "admin";

interface Seed {
  name: string;
  email: string;
  role: Role;
  phone: string;
  address: string;
  city: string;
  orders: number;
}

interface MockOrder {
  id: string;
  number: string;
  placed_at: string;
  status: string;
  total: number;
}

const SAMPLES: Seed[] = [
  { name: "Lan Nguyen", email: "lan@example.com", role: "customer", phone: "0912 345 678", address: "", city: "", orders: 2 },
  { name: "Thu Pham", email: "new@example.com", role: "customer", phone: "0905 111 222", address: "", city: "", orders: 0 },
  { name: "Hoa Tran", email: "many@example.com", role: "customer", phone: "0987 654 321", address: "12 Example Street, Ward 1", city: "Da Nang", orders: 12 },
  { name: "Tuan Vo", email: "slow@example.com", role: "customer", phone: "0933 222 111", address: "", city: "", orders: 2 },
  { name: "Minh", email: "minh@example.com", role: "staff", phone: "0905 111 223", address: "", city: "", orders: 0 },
  { name: "Dealer", email: "dealer@example.com", role: "admin", phone: "0905 111 224", address: "", city: "", orders: 0 },
];

const STATUSES = ["shipped", "delivered", "packed", "confirmed", "placed", "cancelled"];

function makeOrders(count: number): MockOrder[] {
  return Array.from({ length: count }, (_, i) => {
    const n = 24817 - i * 3;
    return {
      id: `doi-${n}`,
      number: `DOI-${n}`,
      placed_at: new Date(Date.UTC(2026, 10, 24 - i, 3, 20)).toISOString(),
      status: STATUSES[i % STATUSES.length],
      total: i === 0 ? 6180000 : 3090000 + (i % 4) * 400000,
    };
  });
}

const accounts = new Map<string, Seed>(SAMPLES.map((s) => [s.email, { ...s }]));
const orders = new Map<string, MockOrder[]>(SAMPLES.map((s) => [s.email, makeOrders(s.orders)]));
let session: string | null = null;
let slowOrderAsks = 0;
let wrongPairs = 0;

const reply = (status: number, body: unknown): Reply => ({ status, body });
const refuse = (status: number, error: string, message: string, errors?: Record<string, string>): Reply =>
  reply(status, errors ? { error, message, errors } : { error, message });

const wait = (ms: number) => new Promise<void>((resolve) => setTimeout(resolve, ms));

/** The sample behaviours an email asks for by how it starts. */
async function behave(email: string): Promise<Reply | null> {
  await wait(email.startsWith("slow") ? 3000 : 60);
  if (email.startsWith("down")) throw new Error("network");
  if (email.startsWith("limit")) return refuse(429, "rate_limited", "Too many tries.");
  return null;
}

/** The signed-in customer's record, or the refusal for a visitor with none (401) or a team session (403). */
function customer(): Seed | Reply {
  const me = session ? accounts.get(session) : undefined;
  if (!me) return refuse(401, "unauthenticated", "Sign in first.");
  return me.role === "customer" ? me : refuse(403, "forbidden", "Not for the shop's team.");
}

const isReply = (v: Seed | Reply): v is Reply => "status" in v;

export async function getMe(): Promise<Reply> {
  await wait(60);
  const me = session ? accounts.get(session) : undefined;
  return me ? reply(200, { email: me.email, name: me.name, role: me.role }) : refuse(401, "unauthenticated", "Sign in first.");
}

export async function signIn(email: string, password: string): Promise<Reply> {
  const key = email.trim().toLowerCase();
  const early = await behave(key);
  if (early) return early;
  const me = accounts.get(key);
  if (!me || password.startsWith("wrong")) return refuse(401, "invalid_credentials", "Wrong email or password.");
  session = key;
  return reply(200, { email: me.email, name: me.name, role: me.role });
}

export async function signOut(): Promise<Reply> {
  await wait(60);
  session = null;
  return reply(204, null);
}

export async function signUp(input: { name: string; email: string; phone: string }): Promise<Reply> {
  const key = input.email.trim().toLowerCase();
  const early = await behave(key);
  if (early) return early;
  if (accounts.has(key)) {
    return refuse(409, "conflict", "That email is taken.", { email: "An account with this email already exists." });
  }
  accounts.set(key, { name: input.name, email: key, role: "customer", phone: input.phone, address: "", city: "", orders: 0 });
  orders.set(key, []);
  session = key;
  return reply(201, { name: input.name, email: key, role: "customer" });
}

export async function getAccount(): Promise<Reply> {
  await wait(60);
  const me = customer();
  if (isReply(me)) return me;
  return reply(200, { name: me.name, email: me.email, phone: me.phone, address: me.address, city: me.city });
}

export async function putAccount(patch: { name: string; phone: string; address: string; city: string }): Promise<Reply> {
  await wait(patch.name.startsWith("Slow") ? 2000 : 60);
  if (patch.name.startsWith("Down")) throw new Error("network");
  const me = customer();
  if (isReply(me)) return me;
  if (patch.phone.replace(/\D/g, "") === "0900000000") {
    return refuse(422, "invalid", "Check the phone number.", { phone: "Check the phone number" });
  }
  Object.assign(me, patch);
  return reply(200, { name: me.name, email: me.email, phone: me.phone, address: me.address, city: me.city });
}

export async function listOrders(page: number, limit: number): Promise<Reply> {
  const me = customer();
  if (isReply(me)) return me;
  const slow = me.email.startsWith("slow") && slowOrderAsks++ === 0;
  await wait(slow ? 3000 : 60);
  if (slow) return refuse(500, "internal", "Something went wrong.");
  const all = orders.get(me.email) ?? [];
  return reply(200, { items: all.slice((page - 1) * limit, page * limit), total: all.length, page, limit });
}

/** A leading 84 stands for 0, as the lookup reads a phone. */
const phoneKey = (phone: string): string => {
  const digits = phone.replace(/\D/g, "");
  return digits.startsWith("84") ? `0${digits.slice(2)}` : digits;
};

export async function addOrder(number: string, phone: string): Promise<Reply> {
  await wait(60);
  const me = customer();
  if (isReply(me)) return me;
  const digits = number.replace(/\D/g, "");
  if (digits === "24801") throw new Error("network");
  if (wrongPairs >= 5) return refuse(429, "rate_limited", "Too many tries.");
  if (digits !== "24818" || phoneKey(phone) !== "0912345678") {
    wrongPairs += 1;
    return refuse(404, "not_found", "Not found.");
  }
  const mine = orders.get(me.email) ?? [];
  const found: MockOrder = { id: "doi-24818", number: "DOI-24818", placed_at: new Date(Date.UTC(2026, 10, 25, 2, 10)).toISOString(), status: "placed", total: 3290000 };
  if (!mine.some((o) => o.id === found.id)) orders.set(me.email, [found, ...mine]);
  return reply(200, found);
}
