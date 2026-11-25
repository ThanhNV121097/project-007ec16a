import { useEffect, useSyncExternalStore } from "react";
<<<<<<< refs/heads/feature/dealer-sign-in-and-protected-admin-area-be-v2
import { ApiError, api } from "./api";
=======
import {
  getMe, postSignIn, postSignOut, type Answer, type Role,
} from "./mock/dealer-sign-in-and-protected-admin-area";
>>>>>>> refs/heads/main

/**
 * WHO IS SIGNED IN, as the shop's server says — and nothing else.
 *
 * The session is the server's: an opaque cookie the browser keeps and this
 * file never sees. What this file holds is the last answer to `GET
 * /v1/auth/me`, so a page that asks "who is this?" gets the server's word, not
 * a note the browser wrote to itself. Nothing is read from or written to
 * browser storage: a forged entry there opens nothing.
 *
 *   useSession()          the visitor signed in, or null. Null while the first
 *                         answer is still awaited, too — a page that must tell
 *                         "still asking" from "nobody" reads useAuth().
 *   useAuth()             { checking, session }: `checking` is true until the
 *                         server has answered once. The gate draws nothing and
 *                         redirects nowhere while it is.
 *   useRole()             "admin", "staff", "customer", or "" for nobody.
 *   signIn(email, password)  resolves "ok", "wrong", "limited" or "unreachable".
 *   signOut()             ends the session on the server and here.
 *
<<<<<<< refs/heads/feature/dealer-sign-in-and-protected-admin-area-be-v2
 * The three calls go to `/v1/auth/*` through src/lib/api.ts, which sends the
 * session cookie with every request.
 */

/** What an account is: the shop's team is admin and staff. */
export type Role = "admin" | "staff" | "customer";

export interface Session {
  email: string;
  name: string;
  /** What the server says this account is. */
=======
 * The three calls go through the mock for now; the backend story points them
 * at `/v1/auth/*` through src/lib/api.ts (credentials: "include") and changes
 * nothing else in this file.
 */
export interface Session {
  email: string;
  name: string;
  /** What the server says this account is: the shop's team is admin and staff. */
>>>>>>> refs/heads/main
  role: Role;
}

export type SignInResult = "ok" | "wrong" | "limited" | "unreachable";

type Phase = "idle" | "checking" | "ready";

interface State {
  phase: Phase;
  session: Session | null;
}

let state: State = { phase: "idle", session: null };
// Bumped by every sign-in and sign-out, so a slower "who am I" answer that
// was asked before one of them can never put the old visitor back.
let epoch = 0;
const listeners = new Set<() => void>();

function set(next: State) {
  state = next;
  listeners.forEach((l) => l());
}

function subscribe(onChange: () => void) {
  listeners.add(onChange);
  return () => {
    listeners.delete(onChange);
  };
}

function snapshot(): State {
  return state;
}

/** The three fields of an answer, or null when the body is not an account. */
function toSession(body: unknown): Session | null {
  const v = body as { email?: unknown; name?: unknown; role?: unknown } | null;
  if (!v || typeof v.email !== "string" || typeof v.name !== "string") return null;
  if (v.role !== "admin" && v.role !== "staff" && v.role !== "customer") return null;
  return { email: v.email, name: v.name, role: v.role };
}

<<<<<<< refs/heads/feature/dealer-sign-in-and-protected-admin-area-be-v2
/** Asks the server who is signed in. Any failure, a 401 or an unreachable server, is nobody. */
=======
/** Asks the server who is signed in. An unreachable server is nobody. */
>>>>>>> refs/heads/main
async function check(): Promise<void> {
  const asked = epoch;
  let found: Session | null = null;
  try {
<<<<<<< refs/heads/feature/dealer-sign-in-and-protected-admin-area-be-v2
    found = toSession(await api.get<unknown>("auth/me"));
=======
    const answer = await getMe();
    if (answer.status === 200) found = toSession(answer.body);
>>>>>>> refs/heads/main
  } catch {
    found = null;
  }
  if (asked === epoch) set({ phase: "ready", session: found });
}

function ensureChecked() {
  if (state.phase !== "idle") return;
  set({ phase: "checking", session: null });
  void check();
}

/**
 * The visitor and whether the server has answered yet. The first component to
 * ask starts the one question; every other reads its answer.
 */
export function useAuth(): { checking: boolean; session: Session | null } {
  const s = useSyncExternalStore(subscribe, snapshot, snapshot);
  useEffect(() => {
    ensureChecked();
  }, []);
  return { checking: s.phase !== "ready", session: s.session };
}

/** The visitor signed in, or null. */
export function useSession(): Session | null {
  return useAuth().session;
}

<<<<<<< refs/heads/feature/dealer-sign-in-and-protected-admin-area-be-v2
/** The word the form acts on for a sign-in the server refused or could not answer. */
function verdict(e: unknown): SignInResult {
  if (e instanceof ApiError) {
    if (e.status === 429) return "limited";
    if (e.status === 401 || e.status === 422) return "wrong";
  }
=======
/** The server's verdict on one sign-in attempt, as the word the form acts on. */
function verdict(answer: Answer): SignInResult {
  if (answer.status === 429) return "limited";
  if (answer.status === 401 || answer.status === 422) return "wrong";
>>>>>>> refs/heads/main
  return "unreachable";
}

/**
 * Signs in with what was typed. On success the session is the server's own
 * answer; on any refusal nothing changes.
 */
export async function signIn(email: string, password: string): Promise<SignInResult> {
<<<<<<< refs/heads/feature/dealer-sign-in-and-protected-admin-area-be-v2
  let body: unknown;
  try {
    body = await api.post<unknown>("auth/sign-in", { email: email.trim(), password });
  } catch (e) {
    return verdict(e);
  }
  const session = toSession(body);
  if (!session) return "unreachable";
=======
  let answer: Answer;
  try {
    answer = await postSignIn(email.trim(), password);
  } catch {
    return "unreachable";
  }
  const session = answer.status === 200 ? toSession(answer.body) : null;
  if (!session) return verdict(answer);
>>>>>>> refs/heads/main
  epoch += 1;
  set({ phase: "ready", session });
  return "ok";
}

/**
 * Signs the visitor out. The screen leaves the area at once; the server is
 * told after, and its answer is always 204, so there is nothing to report if
 * it cannot be reached — the cookie it never heard about ends by itself.
 */
export async function signOut(): Promise<void> {
  epoch += 1;
  set({ phase: "ready", session: null });
  try {
<<<<<<< refs/heads/feature/dealer-sign-in-and-protected-admin-area-be-v2
    await api.post("auth/sign-out");
=======
    await postSignOut();
>>>>>>> refs/heads/main
  } catch {
    // Nothing more can be done from here.
  }
}

/** The role the visitor acts as, or "" when nobody is signed in. */
export function useRole(): Role | "" {
  return useSession()?.role ?? "";
}

/**
 * Whether the visitor may enter an area open to `roles`.
 *
 * An empty list is open to anyone signed in. Nobody signed in is never
 * allowed: a role check is not a sign-in check.
 */
export function allowed(session: Session | null, roles?: readonly string[]): boolean {
  if (!session) return false;
  if (!roles || roles.length === 0) return true;
  return roles.some((r) => r.trim().toLowerCase() === session.role);
}
