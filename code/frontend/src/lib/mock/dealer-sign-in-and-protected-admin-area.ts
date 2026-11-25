/**
 * THE UI STAGE'S STAND-IN FOR `/v1/auth/*` (docs/architecture/services.md
 * §3.7–3.9). Deleted by the backend story, which points src/lib/session.ts at
 * the real endpoints through src/lib/api.ts instead.
 *
 * Each function answers the way the HTTP call will: a status and a body, or a
 * thrown error when the shop cannot be reached. The sample accounts are the
 * three roles; nothing else signs in. The session lives in this module's
 * memory (never in browser storage), so a reload forgets it — the real cookie
 * is what keeps it, and that is proved against the running story.
 *
 *   dealer@example.com      Dealer-Pass1                   admin
 *   minh.le@example.com     correct horse battery staple   staff
 *   lan.nguyen@example.com  correct horse battery staple   customer
 *   down@example.com        any password                   the shop cannot be reached
 *
 * Five wrong tries for one email lock it, as the real limit does.
 */
export type Role = "admin" | "staff" | "customer";

export interface AuthUser {
  email: string;
  name: string;
  role: Role;
}

/** What an HTTP answer carries: its status and its JSON body, if any. */
export interface Answer {
  status: number;
  body: unknown;
}

const PHRASE = "correct horse battery staple";

const ACCOUNTS: readonly (AuthUser & { password: string })[] = [
  { email: "dealer@example.com", name: "Dealer", role: "admin", password: "Dealer-Pass1" },
  { email: "minh.le@example.com", name: "Minh Le", role: "staff", password: PHRASE },
  { email: "lan.nguyen@example.com", name: "Lan Nguyen", role: "customer", password: PHRASE },
];

const UNREACHABLE = "down@example.com";
const LATENCY_MS = 400;
const MAX_FAILURES = 5;

let current: AuthUser | null = null;
const failures = new Map<string, number>();

function after<T>(value: () => T): Promise<T> {
  return new Promise((resolve) => setTimeout(() => resolve(value()), LATENCY_MS));
}

/** POST /v1/auth/sign-in */
export async function postSignIn(email: string, password: string): Promise<Answer> {
  const key = email.trim().toLowerCase();
  if (key === UNREACHABLE) {
    await after(() => null);
    throw new TypeError("Failed to fetch");
  }
  return after(() => {
    if ((failures.get(key) ?? 0) >= MAX_FAILURES) {
      return { status: 429, body: { error: "rate_limited", message: "Too many tries. Please wait 15 minutes." } };
    }
    const account = ACCOUNTS.find((a) => a.email === key);
    if (!account || account.password !== password) {
      failures.set(key, (failures.get(key) ?? 0) + 1);
      return { status: 401, body: { error: "invalid_credentials", message: "Wrong email or password." } };
    }
    current = { email: account.email, name: account.name, role: account.role };
    return { status: 200, body: current };
  });
}

/** GET /v1/auth/me */
export function getMe(): Promise<Answer> {
  return after(() => (current
    ? { status: 200, body: current }
    : { status: 401, body: { error: "unauthenticated", message: "Sign in to continue." } }));
}

/** POST /v1/auth/sign-out: always 204. */
export function postSignOut(): Promise<Answer> {
  return after(() => {
    current = null;
    return { status: 204, body: null };
  });
}
